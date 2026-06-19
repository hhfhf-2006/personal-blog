package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/oauth"
	"personal-blog-backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// GitHubLogin 重定向用户到 GitHub OAuth 授权页面
//
//	GET /api/v1/auth/github?redirect=xxx
//
// redirect 参数（可选）：登录成功后要跳回的页面地址
func (ctrl *Controller) GitHubLogin(c *gin.Context) {
	state, err := oauth.GenerateState()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	// state 存入 cookie（5 分钟有效，SameSite=Lax 防 CSRF），回调时验证
	ctrl.setOAuthCookie(c,"github_oauth_state", state, 300)

	// 将 redirect 参数也保存到 cookie，回调成功后带回
	if redirect := c.Query("redirect"); redirect != "" {
		ctrl.setOAuthCookie(c,"github_oauth_redirect", redirect, 300)
	}

	authURL := ctrl.githubOAuth.GetAuthURL(state)
	c.Redirect(http.StatusFound, authURL)
}

// GitHubCallback 处理 GitHub OAuth 回调
//
//	GET /api/v1/auth/github/callback?code=xxx&state=yyy
//
// 成功后在 service 层自动注册/登录，生成 JWT，
// 最后重定向到前端登录页并携带 token 和用户信息。
func (ctrl *Controller) GitHubCallback(c *gin.Context) {
	// 1. 验证 state（防 CSRF）
	stateCookie, err := c.Cookie("github_oauth_state")
	if err != nil || stateCookie == "" {
		ctrl.redirectWithError(c, "登录会话已过期，请重新登录")
		return
	}
	// 立即清除 cookie，防止复用
	ctrl.setOAuthCookie(c,"github_oauth_state", "", -1)

	state := c.Query("state")
	if state == "" || state != stateCookie {
		ctrl.redirectWithError(c, "登录验证失败，请重试")
		return
	}

	// 2. 获取授权码
	code := c.Query("code")
	if code == "" {
		ctrl.redirectWithError(c, "GitHub 授权失败")
		return
	}

	// 3. 用授权码换取 GitHub 用户信息
	ghUser, err := ctrl.githubOAuth.ExchangeCode(code)
	if err != nil {
		ctrl.redirectWithError(c, fmt.Sprintf("GitHub 登录失败: %v", err))
		return
	}

	// 4. 查找或创建用户，生成 JWT
	result, err := ctrl.userService.LoginByGithub(ghUser)
	if err != nil {
		if apperror.IsBadRequest(err) {
			ctrl.redirectWithError(c, err.Error())
		} else {
			ctrl.redirectWithError(c, "服务器内部错误")
		}
		return
	}

	// 5. 将 token 和 user 编码为 URL Fragment，重定向到前端
	//    使用 Fragment (#) 而非 Query (?)，因为 Fragment 不会被发送到服务器，
	//    从而避免 token 出现在服务器日志、浏览器历史和 Referer 头中。
	userJSON, err := json.Marshal(result.User)
	if err != nil {
		ctrl.redirectWithError(c, "服务器内部错误")
		return
	}

	params := url.Values{}
	params.Set("token", result.Token)
	params.Set("user", string(userJSON))

	// 恢复登录前保存的 redirect 地址
	if redirectCookie, _ := c.Cookie("github_oauth_redirect"); redirectCookie != "" {
		params.Set("redirect", redirectCookie)
		ctrl.setOAuthCookie(c, "github_oauth_redirect", "", -1)
	}

	redirectURL := "/login.html#" + params.Encode()
	c.Redirect(http.StatusFound, redirectURL)
}

// setOAuthCookie 设置带 SameSite=Lax、HttpOnly 的 cookie。
// Secure 标志根据 redirectURI 是否使用 HTTPS 自动决定（本地 HTTP 开发环境自动关闭）。
func (ctrl *Controller) setOAuthCookie(c *gin.Context, name, value string, maxAge int) {
	// 过滤 value 中的危险字符，防止 HTTP 响应头注入（\r\n 可注入额外 Set-Cookie 头，; 可劫持 cookie 属性）
	value = sanitizeCookieValue(value)
	secure := ""
	if ctrl.secureCookie {
		secure = "; Secure"
	}
	cookie := fmt.Sprintf("%s=%s; Path=/; Max-Age=%d; SameSite=Lax; HttpOnly%s", name, value, maxAge, secure)
	c.Writer.Header().Add("Set-Cookie", cookie)
}

// sanitizeCookieValue 移除 cookie 值中可能被用于 HTTP 响应头注入的危险字符
func sanitizeCookieValue(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, ";", "")
	return v
}

// redirectWithError 重定向到登录页并显示错误信息。
// 同时清除 OAuth cookie，防止残留的 state/redirect cookie 影响下次登录。
func (ctrl *Controller) redirectWithError(c *gin.Context, msg string) {
	params := url.Values{}
	params.Set("error", msg)

	// 从 cookie 恢复登录前保存的 redirect 地址（GitHubLogin 中存入）
	if redirectCookie, _ := c.Cookie("github_oauth_redirect"); redirectCookie != "" {
		params.Set("redirect", redirectCookie)
	}
	// 清除 OAuth 相关 cookie，防止错误路径下的 cookie 残留
	ctrl.setOAuthCookie(c, "github_oauth_state", "", -1)
	ctrl.setOAuthCookie(c, "github_oauth_redirect", "", -1)

	c.Redirect(http.StatusFound, "/login.html?"+params.Encode())

	// 阻止后续中间件写入
	c.Abort()
}
