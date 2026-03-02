const STATIC_CACHE = 'ntp-static-v2';
const RUNTIME_CACHE = 'ntp-runtime-v2';
const PRECACHE_URLS = [
    '/',
    '/static/index.html',
    '/static/logo.svg',
    '/static/i18n/en.json',
    '/static/i18n/zh.json'
];

self.addEventListener('install', (event) => {
    event.waitUntil((async () => {
        const cache = await caches.open(STATIC_CACHE);
        await cache.addAll(PRECACHE_URLS);
        await self.skipWaiting();
    })());
});

self.addEventListener('activate', (event) => {
    event.waitUntil((async () => {
        const cacheNames = await caches.keys();
        await Promise.all(
            cacheNames
                .filter((name) => name !== STATIC_CACHE && name !== RUNTIME_CACHE)
                .map((name) => caches.delete(name))
        );
        await self.clients.claim();
    })());
});

self.addEventListener('fetch', (event) => {
    const { request } = event;
    if (request.method !== 'GET') return;

    const url = new URL(request.url);
    if (url.origin !== self.location.origin) return;

    // API 请求仍走网络，避免缓存过期认证态或用户数据
    if (url.pathname.startsWith('/api/')) return;

    // 登录页相关资源强制走网络，降低特定 WebView 命中旧缓存导致崩溃的概率
    if (isLoginPageAsset(url.pathname)) {
        event.respondWith(networkOnlyWithFallback(request));
        return;
    }

    // i18n 清单优先走网络，避免长时间持有旧版本映射
    if (url.pathname === '/static/i18n/manifest.json') {
        event.respondWith(networkFirst(request));
        return;
    }

    // 页面导航优先走网络，失败时回退缓存
    if (request.mode === 'navigate') {
        event.respondWith(networkFirst(request));
        return;
    }

    // 静态资源和本地图标优先读缓存，缺失时再回源
    if (url.pathname.startsWith('/static/') || url.pathname.startsWith('/data/icons/')) {
        event.respondWith(cacheFirst(request));
    }
});

self.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    }
});

async function networkFirst(request) {
    const cache = await caches.open(STATIC_CACHE);
    try {
        const response = await fetch(request);
        if (response && response.ok) {
            await cache.put(request, response.clone());
        }
        return response;
    } catch {
        const cached = await cache.match(request);
        if (cached) return cached;

        const fallback = await cache.match('/');
        if (fallback) return fallback;
        return Response.error();
    }
}

async function cacheFirst(request) {
    const cache = await caches.open(RUNTIME_CACHE);
    const cached = await cache.match(request);
    if (cached) return cached;

    const response = await fetch(request);
    if (response && response.ok) {
        await cache.put(request, response.clone());
    }
    return response;
}

function isLoginPageAsset(pathname) {
    return pathname === '/login' ||
        pathname === '/static/login.html' ||
        pathname === '/static/login.css' ||
        pathname === '/static/login.js';
}

async function networkOnlyWithFallback(request) {
    const staticCache = await caches.open(STATIC_CACHE);
    const runtimeCache = await caches.open(RUNTIME_CACHE);
    try {
        const networkRequest = new Request(request, { cache: 'no-store' });
        const response = await fetch(networkRequest);
        if (response && response.ok) {
            await staticCache.put(request, response.clone());
        }
        return response;
    } catch {
        return (await staticCache.match(request)) ||
            (await runtimeCache.match(request)) ||
            Response.error();
    }
}
