const AUTH_CHECK_ENDPOINT = '/api/auth/check';
const LOGIN_ENDPOINT = '/api/login';

const dom = {
    get form() { return this._form ||= document.getElementById('loginForm'); },
    get username() { return this._username ||= document.getElementById('username'); },
    get password() { return this._password ||= document.getElementById('password'); },
    get loginBtn() { return this._loginBtn ||= document.getElementById('loginBtn'); },
    get error() { return this._error ||= document.getElementById('loginError'); },
    get languageSelect() { return this._languageSelect ||= document.getElementById('languageSelect'); }
};

async function safeParseJSON(response) {
    try {
        return await response.json();
    } catch (error) {
        return null;
    }
}

function showLoginError(message) {
    dom.error.textContent = message;
    dom.error.classList.add('show');
}

function clearLoginError() {
    dom.error.classList.remove('show');
    dom.error.textContent = '';
}

function setLoginButtonState(isLoading, textKey = 'login.logging') {
    dom.loginBtn.disabled = isLoading;
    if (isLoading) {
        dom.loginBtn.innerHTML = `<span data-i18n="${textKey}">${i18n.t(textKey)}</span>`;
    }
}

async function checkAuth() {
    try {
        const response = await fetch(AUTH_CHECK_ENDPOINT, {
            headers: i18n.getRequestHeaders()
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

    const username = dom.username.value.trim();
    const password = dom.password.value;

    if (!username || !password) {
        showLoginError(i18n.t('login.invalidCredentials'));
        return;
    }

    const originalText = dom.loginBtn.innerHTML;
    let loginSucceeded = false;

    clearLoginError();
    setLoginButtonState(true);

    try {
        const response = await fetch(LOGIN_ENDPOINT, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                ...i18n.getRequestHeaders()
            },
            body: JSON.stringify({ username, password })
        });

        const data = await safeParseJSON(response);

        if (response.ok && data?.success) {
            loginSucceeded = true;
            dom.loginBtn.innerHTML = `<span data-i18n="login.success">${i18n.t('login.success')}</span>`;
            setTimeout(() => {
                window.location.href = '/';
            }, 200);
            return;
        }

        showLoginError(data?.message ?? i18n.t('login.failed'));
    } catch (error) {
        console.error('Login error:', error);
        showLoginError(i18n.t('login.failed'));
    } finally {
        if (!loginSucceeded) {
            dom.loginBtn.innerHTML = originalText;
            setLoginButtonState(false);
        }
    }
}

function initLanguageSelector() {
    if (!dom.languageSelect) {
        return;
    }

    dom.languageSelect.value = i18n.getCurrentLocale();
    dom.languageSelect.addEventListener('change', async ({ target }) => {
        await i18n.setLocale(target.value);
    });
}

document.addEventListener('DOMContentLoaded', async () => {
    await i18n.init();
    initLanguageSelector();
    await checkAuth();
    dom.form?.addEventListener('submit', handleLogin);
});
