/* ============================================================
   API 工具函数 — 封装 fetch，自动携带 JWT 令牌
   ============================================================ */

const API_BASE = '/api/v1';

/**
 * 发送 API 请求，自动从 localStorage 读取 token 并附加到 Authorization 头。
 * 默认 15 秒超时，防止后端卡死时前端无限等待。
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

  // 请求超时控制：默认 15 秒，可通过 options.timeout 覆盖
  const timeout = options.timeout || 15000;
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const res = await fetch(API_BASE + path, {
      ...options,
      headers,
      signal: controller.signal,
    });

    let data;
    try {
      data = await res.json();
    } catch (jsonErr) {
      // 后端返回了非 JSON 内容（如 HTML 错误页、网关超时页等）
      const err = new Error('服务器返回了无效的响应，请稍后重试');
      err.status = res.status || 502;
      throw err;
    }

    if (!res.ok) {
      // JWT 过期或无效 → 清除登录状态并跳转到登录页
      if (res.status === 401 && token) {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        clearGameData(); // 会话失效同样清理游戏数据，防止泄漏给下一个账号
        // 防止在登录页上重复跳转
        if (!window.location.pathname.endsWith('/login.html')) {
          const currentURL = encodeURIComponent(window.location.pathname + window.location.search);
          window.location.href = '/login.html?redirect=' + currentURL;
        }
      }

      const err = new Error(data.msg || '请求失败');
      err.status = res.status;
      err.data = data;
      throw err;
    }

    return data;
  } catch (err) {
    if (err.name === 'AbortError') {
      const timeoutErr = new Error('请求超时，请检查网络后重试');
      timeoutErr.status = 408;
      throw timeoutErr;
    }
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}

// 便捷方法
api.get = (path) => api(path, { method: 'GET' });
api.post = (path, body) => api(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined });
api.put = (path, body) => api(path, { method: 'PUT', body: body !== undefined ? JSON.stringify(body) : undefined });
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

/* ------------------------------------------------------------
   游戏数据清理 —— 防止账号之间的游戏数据泄漏
   ------------------------------------------------------------ */

// 所有与游戏相关的存储 key（均非用户专属，登出 / 切换账号时必须全部清除）
const GAME_LOCAL_KEYS = ['bestScore', 'gameState', 'pendingScore', 'pendingBoardRefresh'];
const GAME_SESSION_KEYS = ['2048_session_active'];

/**
 * 彻底清除浏览器中所有与游戏相关的存储（localStorage + sessionStorage）。
 * 用于登出、会话失效以及登录/切换账号，确保 A 的数据不会泄漏给 B。
 */
function clearGameData() {
  try {
    GAME_LOCAL_KEYS.forEach(function (k) { localStorage.removeItem(k); });
  } catch (e) {}
  try {
    GAME_SESSION_KEYS.forEach(function (k) { sessionStorage.removeItem(k); });
  } catch (e) {}
}

/**
 * 销毁会话前的"分数结算预检"：把本地暂存的待提交分数用 keepalive fetch 送达后端。
 * 仅在仍持有 token 时执行（登出时 token 尚未清除）。keepalive 保证即便随后立即
 * 跳转/清空存储，请求也能发完。
 *
 * 注意：用 keepalive fetch 而非 sendBeacon —— 后端用 Authorization 头鉴权，
 * sendBeacon 无法自定义请求头会被 401 拒绝。
 */
let _gameScoreFlushed = 0; // 本会话已结算过的最高分，去重防抖
function flushGameScore() {
  let token;
  try { token = localStorage.getItem('token'); } catch (e) { return; }
  if (!token) return;

  function readScore(key) {
    try {
      const obj = JSON.parse(localStorage.getItem(key) || 'null');
      return obj && obj.score > 0 ? obj.score : 0;
    } catch (e) { return 0; }
  }

  const score = Math.max(readScore('pendingScore'), readScore('gameState'));
  if (score <= 0 || score <= _gameScoreFlushed) return;
  _gameScoreFlushed = score;

  try {
    fetch(API_BASE + '/games/scores', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + token },
      body: JSON.stringify({ game_name: '2048', score: score }),
      keepalive: true
    }).catch(function () {});
  } catch (e) {}
}

/** 保存登录信息 */
function saveAuth(token, user) {
  // 登录 / 切换账号前，先清除上一账号可能残留的游戏数据，防止跨账号泄漏。
  // 此处不做分数结算 —— 残留分数应在上一账号登出时已结算；若用此刻的新 token
  // 提交旧分，会把 A 的成绩错记到 B 名下。
  clearGameData();
  localStorage.setItem('token', token);
  localStorage.setItem('user', JSON.stringify(user));
}

/** 清除登录状态 */
function logout() {
  // 1. 预检结算：销毁会话前先把当前分数送达后端（此时 token 尚未清除）
  flushGameScore();
  // 2. 核心修复：彻底清除所有游戏相关存储，防止泄漏给下一个登录账号
  clearGameData();
  // 3. 清除认证状态后跳转
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

/** 当前用户是否为管理员 */
function isAdmin() {
  const user = getUser();
  return user && user.is_admin === true;
}

/** 更新页面头部：根据登录状态切换导航链接 */
function updateHeader() {
  const nav = document.getElementById('navLinks');
  if (!nav) return;

  const user = getUser();
  const is2048Page = window.location.pathname.endsWith('/2048.html');
  const isLeaderboardPage = window.location.pathname.endsWith('/leaderboard.html');
  const isGameRelated = is2048Page || isLeaderboardPage;

  // 游戏中心下拉菜单 HTML
  const gameDropdown = `
    <li class="nav-dropdown" id="gameDropdown">
      <button class="nav-dropdown-toggle${isGameRelated ? ' active' : ''}" aria-haspopup="true" aria-expanded="false">
        游戏中心
        <svg class="dropdown-arrow" width="10" height="6" viewBox="0 0 10 6" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 1 5 5 9 1"/></svg>
      </button>
      <ul class="nav-dropdown-menu">
        <li><a href="/2048.html"${is2048Page ? ' class="active"' : ''}>2048</a></li>
        <li><a href="/leaderboard.html"${isLeaderboardPage ? ' class="active"' : ''}>排行榜</a></li>
      </ul>
    </li>`;

  if (user) {
    const newPostLink = user.is_admin
      ? `<li><a href="/new-post.html" class="nav-new-post">写文章</a></li>`
      : '';

    const usernameDisplay = user.is_admin
      ? `<li><a href="/admin-users.html" style="color:var(--accent);font-size:var(--font-size-sm);font-weight:500;">
          ${escapeHTML(user.username)}
        </a></li>`
      : `<li><a href="/profile.html" title="个人中心">${escapeHTML(user.username)}</a></li>`;

    nav.innerHTML = `
      <li><a href="/">首页</a></li>
      ${gameDropdown}
      ${newPostLink}
      ${usernameDisplay}
      <li><button class="btn-logout" onclick="logout()">退出</button></li>
    `;
  } else {
    nav.innerHTML = `
      <li><a href="/">首页</a></li>
      ${gameDropdown}
      <li><a href="/login.html">登录</a></li>
      <li><a href="/register.html">注册</a></li>
    `;
  }

  // 绑定下拉菜单事件
  initGameDropdown();
}

/** 初始化游戏中心下拉菜单 */
function initGameDropdown() {
  const dropdown = document.getElementById('gameDropdown');
  if (!dropdown || dropdown.dataset.initialized) return;
  dropdown.dataset.initialized = '1';

  const toggle = dropdown.querySelector('.nav-dropdown-toggle');
  const menu = dropdown.querySelector('.nav-dropdown-menu');

  toggle.addEventListener('click', function (e) {
    e.preventDefault();
    e.stopPropagation();
    const isOpen = menu.classList.contains('open');
    // 关闭所有下拉菜单
    document.querySelectorAll('.nav-dropdown-menu.open').forEach(m => m.classList.remove('open'));
    document.querySelectorAll('.nav-dropdown-toggle').forEach(t => t.setAttribute('aria-expanded', 'false'));
    if (!isOpen) {
      menu.classList.add('open');
      toggle.setAttribute('aria-expanded', 'true');
    }
  });

  // 点击菜单项时关闭下拉
  menu.querySelectorAll('a').forEach(link => {
    link.addEventListener('click', function () {
      menu.classList.remove('open');
      toggle.setAttribute('aria-expanded', 'false');
    });
  });
}

// 全局点击关闭下拉菜单
document.addEventListener('click', function (e) {
  if (!e.target.closest('.nav-dropdown')) {
    document.querySelectorAll('.nav-dropdown-menu.open').forEach(m => m.classList.remove('open'));
    document.querySelectorAll('.nav-dropdown-toggle').forEach(t => t.setAttribute('aria-expanded', 'false'));
  }
});

/* ============================================================
   通用工具
   ============================================================ */

/** HTML 转义（防 XSS） */
function escapeHTML(str) {
  if (str == null) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

/** 格式化日期时间（年-月-日 时:分），使用北京时间（UTC+8） */
function formatDate(dateStr) {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    const parts = new Intl.DateTimeFormat('zh-CN', {
      timeZone: 'Asia/Shanghai',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    }).formatToParts(d);
    const get = function(type) {
      var found = null;
      for (var i = 0; i < parts.length; i++) {
        if (parts[i].type === type) { found = parts[i].value; break; }
      }
      return found || '00';
    };
    return get('year') + '-' + get('month') + '-' + get('day') + ' ' + get('hour') + ':' + get('minute');
  } catch (e) {
    // 降级：回退到本地时区
    var fallback = new Date(dateStr);
    var y = fallback.getFullYear();
    var m = String(fallback.getMonth() + 1).padStart(2, '0');
    var day = String(fallback.getDate()).padStart(2, '0');
    var h = String(fallback.getHours()).padStart(2, '0');
    var min = String(fallback.getMinutes()).padStart(2, '0');
    return y + '-' + m + '-' + day + ' ' + h + ':' + min;
  }
}

/** 格式化日期时间（月-日 时:分），使用北京时间（UTC+8），用于排行榜等场景 */
function formatDateTime(dateStr) {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    const parts = new Intl.DateTimeFormat('zh-CN', {
      timeZone: 'Asia/Shanghai',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    }).formatToParts(d);
    const get = function(type) {
      for (var i = 0; i < parts.length; i++) {
        if (parts[i].type === type) return parts[i].value;
      }
      return '00';
    };
    return get('month') + '-' + get('day') + ' ' + get('hour') + ':' + get('minute');
  } catch (e) {
    var fallback = new Date(dateStr);
    var m = String(fallback.getMonth() + 1).padStart(2, '0');
    var day = String(fallback.getDate()).padStart(2, '0');
    var h = String(fallback.getHours()).padStart(2, '0');
    var min = String(fallback.getMinutes()).padStart(2, '0');
    return m + '-' + day + ' ' + h + ':' + min;
  }
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

/**
 * 安全渲染 Markdown：先用 marked 解析为 HTML，再用 DOMPurify 移除危险标签。
 * 仅允许安全的 HTML 标签和属性，拦截 <script>、onerror 等 XSS 攻击向量。
 *
 * 额外支持 LaTeX 数学公式：
 *   - $...$  行内公式（inline）
 *   - $$...$$ 跨行公式（display / block）
 * 公式通过 KaTeX 渲染为 HTML。如果 KaTeX 未加载，则显示为转义后的原始公式文本。
 *
 * 降级策略（按优先级）：
 *   1. marked + DOMPurify 都可用 → 解析为 HTML 并净化（最佳体验）
 *   2. marked 可用、DOMPurify 不可用 → 直接 HTML 转义原始 Markdown（安全但无格式）
 *   3. marked 不可用 → 直接 HTML 转义原始 Markdown（纯文本兜底）
 */
/**
 * 注册 marked 扩展：==文本== → <mark class="md-mark">文本</mark>（高亮块）。
 * 作为内联（inline）扩展，自动跳过代码块/行内代码，且支持高亮内部再嵌套
 * 粗体、链接等内联语法。仅在 marked 已加载时注册（部分页面不引入 marked）。
 */
if (typeof marked !== 'undefined' && typeof marked.use === 'function') {
  marked.use({
    extensions: [{
      name: 'highlightMark',
      level: 'inline',
      start(src) { return src.indexOf('=='); },
      tokenizer(src) {
        // ==非空白开头 …… 非空白结尾==，惰性匹配，避免吞掉后续内容
        const match = /^==(?=\S)([\s\S]*?\S)==/.exec(src);
        if (match) {
          return {
            type: 'highlightMark',
            raw: match[0],
            text: match[1],
            tokens: this.lexer.inlineTokens(match[1]),
          };
        }
      },
      renderer(token) {
        return '<mark class="md-mark">' + this.parser.parseInline(token.tokens) + '</mark>';
      },
    }],
  });
}

function renderMarkdown(raw) {
  if (typeof marked === 'undefined') {
    // 降级：marked 未加载，纯文本显示
    return escapeHTML(raw);
  }
  if (typeof DOMPurify === 'undefined') {
    // 降级：DOMPurify 不可用（CDN 故障等），不能安全渲染 HTML
    // 直接转义原始 Markdown 文本，防止 XSS 攻击
    return escapeHTML(raw);
  }

  // ── Phase 1: 提取数学公式 ──────────────────────────────────
  var displayMath = [];   // $$...$$ 跨行公式
  var inlineMath = [];    // $...$   行内公式

  // 先提取跨行公式 $$...$$（必须在内联 $...$ 之前，避免第一个 $ 被误匹配）
  // 占位符中放入公式原文（HTML 转义），作为 KaTeX 加载失败时的可见降级
  var processed = raw.replace(/\$\$([\s\S]*?)\$\$/g, function(_, formula) {
    var f = formula.trim();
    displayMath.push(f);
    return '\n<div class="math-disp">' + escapeHTML(f) + '</div>\n';
  });

  // 再提取行内公式 $...$（跳过行首/行尾空白，且不匹配 $$ 残余）
  // 模式：$ 后面不能紧跟空格（避免误匹配空公式），不能紧跟 $；允许反斜杠（\）开头的 LaTeX 命令
  processed = processed.replace(/(^|[^\\\w$])\$([^$\s][^$]*?)\$/g, function(_, before, formula) {
    var f = formula.trim();
    inlineMath.push(f);
    return before + '<span class="math-inl">' + escapeHTML(f) + '</span>';
  });

  // ── Phase 2: Markdown → HTML → 净化 ────────────────────────
  var rawHTML = marked.parse(processed);
  var cleanHTML = DOMPurify.sanitize(rawHTML, {
    ALLOWED_TAGS: [
      'h1','h2','h3','h4','h5','h6','p','br','hr',
      'ul','ol','li','blockquote','pre','code','em','strong','del','mark','u','ins','s',
      'a','img','table','thead','tbody','tr','th','td',
      'div','span','kbd','sup','sub','details','summary',
      'input','label','dl','dt','dd',
    ],
    ALLOWED_ATTR: ['href','src','alt','title','class','id','target','rel','type','checked','disabled','data-midx'],
  });

  // ── Phase 3: KaTeX 渲染数学公式（DOM 顺序匹配，不依赖自定义属性） ──
  // 用 class 选择所有占位符，按 DOM 出现顺序依次计数匹配。
  // 占位符中已有公式原文（HTML 转义），渲染失败时自动作为可见降级。
  var container = document.createElement('div');
  container.innerHTML = cleanHTML;

  var placeholders = container.querySelectorAll('.math-disp, .math-inl');
  var dispCount = 0;
  var inlCount = 0;

  for (var p = 0; p < placeholders.length; p++) {
    var el = placeholders[p];
    var isDisplay = el.classList.contains('math-disp');
    var formula;

    // 按 DOM 出现顺序从对应数组中取公式（顺序保证一致）
    if (isDisplay) {
      formula = displayMath[dispCount];
      dispCount++;
    } else {
      formula = inlineMath[inlCount];
      inlCount++;
    }

    if (formula === undefined) continue;

    var rendered;
    if (typeof katex !== 'undefined') {
      try {
        rendered = katex.renderToString(formula, {
          displayMode: isDisplay,
          throwOnError: false,
          trust: false,
          strict: 'warn'
        });
      } catch (e) {
        // 渲染失败 → 保留占位符中的公式原文作为降级
        continue;
      }
    } else {
      // KaTeX 未加载 → 保留占位符中的公式原文作为降级
      continue;
    }

    // outerHTML 赋值：将占位符元素原地替换为 KaTeX 渲染产物
    el.outerHTML = rendered;
  }

  return container.innerHTML;
}

/** 从路径中提取文章 ID（兼容 /post.html?id=xxx 和 /post.html/xxx） */
function getPostIDFromURL() {
  return getQueryParam('id');
}

/** 搜索表单提交：跳转到首页并携带搜索关键词 */
function handleSearch(e) {
  e.preventDefault();
  const q = document.getElementById('searchInput').value.trim();
  if (q) {
    window.location.href = '/?q=' + encodeURIComponent(q);
  }
}
