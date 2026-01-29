// Login page functionality

// Check if user is already authenticated
async function checkAuth() {
    try {
        const response = await fetch('/api/auth/check');
        const data = await response.json();

        if (data.authenticated) {
            // User is already logged in, redirect to home
            window.location.href = '/';
        }
    } catch (error) {
        console.error('Failed to check auth status:', error);
    }
}

// Handle login form submission
async function handleLogin(event) {
    event.preventDefault();

    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const loginBtn = document.getElementById('loginBtn');
    const errorDiv = document.getElementById('loginError');

    // Disable button and show loading
    loginBtn.disabled = true;
    const originalText = loginBtn.innerHTML;
    loginBtn.innerHTML = `<span data-i18n="login.logging">${i18n.t('login.logging')}</span>`;
    errorDiv.classList.remove('show');

    try {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password }),
        });

        const data = await response.json();

        if (data.success) {
            // Login successful, show success message and redirect
            loginBtn.innerHTML = `<span data-i18n="login.success">${i18n.t('login.success')}</span>`;

            // Wait a short delay to ensure cookie is set before redirect
            setTimeout(() => {
                window.location.href = '/';
            }, 200);
        } else {
            // Login failed, show error
            errorDiv.textContent = data.message;
            errorDiv.classList.add('show');
            loginBtn.disabled = false;
            loginBtn.innerHTML = originalText;
        }
    } catch (error) {
        console.error('Login error:', error);
        errorDiv.textContent = 'Login failed. Please try again.';
        errorDiv.classList.add('show');
        loginBtn.disabled = false;
        loginBtn.innerHTML = originalText;
    }
}

// Initialize language selector
function initLanguageSelector() {
    const languageSelect = document.getElementById('languageSelect');

    // Set initial value
    languageSelect.value = i18n.getCurrentLocale();

    languageSelect.addEventListener('change', async (e) => {
        const newLocale = e.target.value;
        await i18n.setLocale(newLocale);
    });
}

// Initialize page
document.addEventListener('DOMContentLoaded', async () => {
    // Initialize i18n
    await i18n.init();

    // Initialize language selector
    initLanguageSelector();

    // Check authentication status
    await checkAuth();

    // Setup form submission
    document.getElementById('loginForm').addEventListener('submit', handleLogin);
});
