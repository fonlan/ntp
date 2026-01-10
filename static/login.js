// Login page functionality
let currentLocale = 'en';

// Load translations
async function loadTranslations() {
    try {
        const response = await fetch(`/static/i18n/${currentLocale}.json`);
        const translations = await response.json();
        return translations;
    } catch (error) {
        console.error('Failed to load translations:', error);
        return {};
    }
}

// Translate function
function t(key, translations) {
    const keys = key.split('.');
    let value = translations;
    for (const k of keys) {
        value = value?.[k];
    }
    return value || key;
}

// Apply translations to the page
async function applyTranslations() {
    const translations = await loadTranslations();

    // Update elements with data-i18n attribute
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        const translation = t(key, translations);
        if (translation && translation !== key) {
            el.textContent = translation;
        }
    });

    // Update placeholders
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.getAttribute('data-i18n-placeholder');
        const translation = t(key, translations);
        if (translation && translation !== key) {
            el.setAttribute('placeholder', translation);
        }
    });

    // Update titles
    document.querySelectorAll('[data-i18n-title]').forEach(el => {
        const key = el.getAttribute('data-i18n-title');
        const translation = t(key, translations);
        if (translation && translation !== key) {
            el.setAttribute('title', translation);
        }
    });
}

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
    loginBtn.innerHTML = '<span data-i18n="login.logging">Logging in...</span>';
    errorDiv.style.display = 'none';

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
            // Login successful, redirect to home
            window.location.href = '/';
        } else {
            // Login failed, show error
            errorDiv.textContent = data.message;
            errorDiv.style.display = 'block';
            loginBtn.disabled = false;
            loginBtn.innerHTML = originalText;
        }
    } catch (error) {
        console.error('Login error:', error);
        errorDiv.textContent = 'Login failed. Please try again.';
        errorDiv.style.display = 'block';
        loginBtn.disabled = false;
        loginBtn.innerHTML = originalText;
    }
}

// Initialize language selector
function initLanguageSelector() {
    const languageSelect = document.getElementById('languageSelect');

    // Load saved language preference
    const savedLang = localStorage.getItem('ntp_language');
    if (savedLang) {
        currentLocale = savedLang;
        languageSelect.value = savedLang;
    }

    languageSelect.addEventListener('change', (e) => {
        currentLocale = e.target.value;
        localStorage.setItem('ntp_language', currentLocale);
        applyTranslations();
    });
}

// Initialize page
document.addEventListener('DOMContentLoaded', () => {
    // Check authentication status
    checkAuth();

    // Apply translations
    applyTranslations();

    // Initialize language selector
    initLanguageSelector();

    // Setup form submission
    document.getElementById('loginForm').addEventListener('submit', handleLogin);
});
