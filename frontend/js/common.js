/* ============================================================
   API 工具函数 — 封装 fetch，自动携带 JWT 令牌
   ============================================================ */

const API_BASE = '/api/v1';

/**
 * 发送 API 请求，自动从 localStorage 读取 token 并附加到 Authorization 头
 */
async function api(path, options = {}) {
  const token = localStorage.getItem('token');
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(API_BASE + path, {
    ...options,
    headers,
  });

  const data = await res.json();

  if (!res.ok) {
    const err = new Error(data.msg || '请求失败');
    err.status = res.status;
    err.data = data;
    throw err;
  }

  return data;
}

// 便捷方法
api.get = (path) => api(path, { method: 'GET' });
api.post = (path, body) => api(path, { method: 'POST', body: JSON.stringify(body) });
api.put = (path, body) => api(path, { method: 'PUT', body: JSON.stringify(body) });
api.delete = (path) => api(path, { method: 'DELETE' });

/* ============================================================
   认证状态管理
   ============================================================ */

/** 获取当前登录用户信息（从 localStorage） */
function getUser() {
  try {
    const raw = localStorage.getItem('user');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

/** 保存登录信息 */
function saveAuth(token, user) {
  localStorage.setItem('token', token);
  localStorage.setItem('user', JSON.stringify(user));
}

/** 清除登录状态 */
function logout() {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
  window.location.href = '/login.html';
}

/** 获取登录后应该跳回的地址 */
function getRedirectURL() {
  return getQueryParam('redirect') || '/';
}

/** 跳转到登录页，并携带当前页面作为回调地址 */
function goToLogin() {
  const currentURL = encodeURIComponent(window.location.pathname + window.location.search);
  window.location.href = '/login.html?redirect=' + currentURL;
}

/** 是否已登录 */
function isLoggedIn() {
  return !!localStorage.getItem('token');
}

/** 更新页面头部：根据登录状态切换导航链接 */
function updateHeader() {
  const nav = document.getElementById('navLinks');
  if (!nav) return;

  const user = getUser();

  if (user) {
    nav.innerHTML = `
      <li><a href="/">首页</a></li>
      <li><a href="/new-post.html" class="nav-new-post">写文章</a></li>
      <li><span style="color:var(--text-muted);font-size:var(--font-size-sm);">
        ${escapeHTML(user.username)}
      </span></li>
      <li><button class="btn-logout" onclick="logout()">退出</button></li>
    `;
  } else {
    nav.innerHTML = `
      <li><a href="/">首页</a></li>
      <li><a href="/login.html">登录</a></li>
      <li><a href="/register.html">注册</a></li>
    `;
  }
}

/* ============================================================
   通用工具
   ============================================================ */

/** HTML 转义（防 XSS） */
function escapeHTML(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

/** 格式化日期 */
function formatDate(dateStr) {
  const d = new Date(dateStr);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

/** 显示 Toast 消息 */
function showToast(msg, type) {
  type = type || 'success';
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.textContent = msg;
  document.body.appendChild(toast);
  setTimeout(() => toast.remove(), 3000);
}

/** 获取 URL 参数 */
function getQueryParam(name) {
  const params = new URLSearchParams(window.location.search);
  return params.get(name);
}

/** 从路径中提取文章 ID（兼容 /post.html?id=xxx 和 /post.html/xxx） */
function getPostIDFromURL() {
  return getQueryParam('id');
}
