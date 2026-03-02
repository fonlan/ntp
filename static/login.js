const AUTH_CHECK_ENDPOINT = '/api/auth/check';
const LOGIN_ENDPOINT = '/api/login';

const SUPPORTED_LOCALES = ['en', 'zh'];
const LOCALE_MESSAGES = {
    en: {
        pageTitle: 'Login - NTP',
        appTitle: 'NTP - Bookmark Navigation',
        subtitle: 'Please login to continue',
        username: 'Username',
        password: 'Password',
        submit: 'Login',
        logging: 'Logging in...',
        success: 'Success',
        failed: 'Login failed',
        invalidCredentials: 'Please enter username and password',
        language: 'Language'
    },
    zh: {
        pageTitle: '登录 - NTP',
        appTitle: 'NTP - 书签导航',
        subtitle: '请先登录后继续',
        username: '用户名',
        password: '密码',
        submit: '登录',
        logging: '登录中...',
        success: '登录成功',
        failed: '登录失败',
        invalidCredentials: '请输入用户名和密码',
        language: '语言'
    }
};

const state = {
    currentLocale: 'en',
    buttonMode: 'idle'
};

const dom = {
    get form() { return this._form ||= document.getElementById('loginForm'); },
    get username() { return this._username ||= document.getElementById('username'); },
    get password() { return this._password ||= document.getElementById('password'); },
    get loginBtn() { return this._loginBtn ||= document.getElementById('loginBtn'); },
    get loginBtnText() { return this._loginBtnText ||= document.getElementById('loginBtnText'); },
    get error() { return this._error ||= document.getElementById('loginError'); },
    get languageSelect() { return this._languageSelect ||= document.getElementById('languageSelect'); },
    get pageTitle() { return this._pageTitle ||= document.getElementById('pageTitle'); },
    get headerTitle() { return this._headerTitle ||= document.getElementById('loginHeaderTitle'); },
    get subtitle() { return this._subtitle ||= document.getElementById('loginSubtitle'); },
    get usernameLabel() { return this._usernameLabel ||= document.getElementById('usernameLabel'); },
    get passwordLabel() { return this._passwordLabel ||= document.getElementById('passwordLabel'); },
    get languageLabel() { return this._languageLabel ||= document.getElementById('languageLabel'); }
};

function t(key) {
    return LOCALE_MESSAGES[state.currentLocale]?.[key] ?? LOCALE_MESSAGES.en[key] ?? key;
}

function resolveInitialLocale() {
    const savedLocale = localStorage.getItem('ntp-language');
    if (savedLocale && SUPPORTED_LOCALES.includes(savedLocale)) {
        return savedLocale;
    }

    const browserLang = navigator.language || navigator.userLanguage || 'en';
    return browserLang.startsWith('zh') ? 'zh' : 'en';
}

function renderLoginButtonText() {
    if (!dom.loginBtnText) return;

    switch (state.buttonMode) {
        case 'logging':
            dom.loginBtnText.textContent = t('logging');
            break;
        case 'success':
            dom.loginBtnText.textContent = t('success');
            break;
        default:
            dom.loginBtnText.textContent = t('submit');
            break;
    }
}

function applyTranslations() {
    const pageTitle = t('pageTitle');
    document.documentElement.lang = state.currentLocale;
    document.title = pageTitle;

    if (dom.pageTitle) dom.pageTitle.textContent = pageTitle;
    if (dom.headerTitle) dom.headerTitle.textContent = t('appTitle');
    if (dom.subtitle) dom.subtitle.textContent = t('subtitle');
    if (dom.usernameLabel) dom.usernameLabel.textContent = t('username');
    if (dom.passwordLabel) dom.passwordLabel.textContent = t('password');
    if (dom.languageLabel) dom.languageLabel.textContent = t('language');

    renderLoginButtonText();
}

function setLocale(locale) {
    if (!SUPPORTED_LOCALES.includes(locale)) {
        return;
    }

    state.currentLocale = locale;
    localStorage.setItem('ntp-language', locale);
    applyTranslations();
}

function getRequestHeaders() {
    return {
        'X-Locale': state.currentLocale
    };
}

async function safeParseJSON(response) {
    try {
        return await response.json();
    } catch {
        return null;
    }
}

function showLoginError(message) {
    if (!dom.error) return;
    dom.error.textContent = message;
    dom.error.classList.add('show');
}

function clearLoginError() {
    if (!dom.error) return;
    dom.error.classList.remove('show');
    dom.error.textContent = '';
}

function setLoginButtonState(isLoading, mode = 'logging') {
    if (!dom.loginBtn) return;
    dom.loginBtn.disabled = isLoading;
    state.buttonMode = isLoading ? mode : 'idle';
    renderLoginButtonText();
}

function initLanguageSelector() {
    if (!dom.languageSelect) {
        return;
    }

    dom.languageSelect.value = state.currentLocale;
    dom.languageSelect.addEventListener('change', ({ target }) => {
        setLocale(target.value);
    });
}

async function checkAuth() {
    try {
        const response = await fetch(AUTH_CHECK_ENDPOINT, {
            headers: getRequestHeaders()
        });

        if (!response.ok) {
            return;
        }

        const data = await safeParseJSON(response);
        if (data?.authenticated) {
            window.location.href = '/';
        }
    } catch (error) {
        console.error('Failed to check auth status:', error);
    }
}

async function handleLogin(event) {
    event.preventDefault();

    const username = dom.username?.value.trim() || '';
    const password = dom.password?.value || '';

    if (!username || !password) {
        showLoginError(t('invalidCredentials'));
        return;
    }

    clearLoginError();
    setLoginButtonState(true, 'logging');

    try {
        const response = await fetch(LOGIN_ENDPOINT, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                ...getRequestHeaders()
            },
            body: JSON.stringify({ username, password })
        });

        const data = await safeParseJSON(response);

        if (response.ok && data?.success) {
            state.buttonMode = 'success';
            renderLoginButtonText();
            setTimeout(() => {
                window.location.href = '/';
            }, 200);
            return;
        }

        showLoginError(data?.message ?? t('failed'));
    } catch (error) {
        console.error('Login error:', error);
        showLoginError(t('failed'));
    } finally {
        if (state.buttonMode !== 'success') {
            setLoginButtonState(false);
        }
    }
}

async function cleanupServiceWorkerForLogin() {
    if (!('serviceWorker' in navigator)) {
        return false;
    }

    let hadRegistrations = false;
    try {
        const registrations = await navigator.serviceWorker.getRegistrations();
        hadRegistrations = registrations.length > 0;
        await Promise.all(registrations.map(registration => registration.unregister()));
    } catch {
        // 清理失败时继续登录流程
    }

    if ('caches' in window) {
        try {
            const keys = await caches.keys();
            await Promise.all(keys.map(key => caches.delete(key)));
        } catch {
            // 清理失败时继续登录流程
        }
    }

    return hadRegistrations;
}

document.addEventListener('DOMContentLoaded', async () => {
    const hadRegistrations = await cleanupServiceWorkerForLogin();
    const shouldReload = hadRegistrations &&
        !sessionStorage.getItem('ntp-login-sw-cleaned') &&
        'serviceWorker' in navigator &&
        !!navigator.serviceWorker.controller;
    if (shouldReload) {
        sessionStorage.setItem('ntp-login-sw-cleaned', '1');
        window.location.reload();
        return;
    }
    sessionStorage.removeItem('ntp-login-sw-cleaned');

    state.currentLocale = resolveInitialLocale();
    initLanguageSelector();
    applyTranslations();

    await checkAuth();
    dom.form?.addEventListener('submit', handleLogin);
});
