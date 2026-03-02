// i18n internationalization utility
class I18n {
  constructor() {
    this.currentLocale = 'en';
    this.messages = {};
    this.supportedLocales = ['en', 'zh'];
    this.messageCache = new Map();
    this.localeManifest = null;
    this.localeManifestPromise = null;
    this.missingKeys = new Set();
    this.initPromise = null;
    this.ready = false;
  }

  async init() {
    // Avoid duplicate initialization
    if (this.ready) {
      return;
    }

    if (this.initPromise) {
      return this.initPromise;
    }

    this.initPromise = (async () => {
      // Get saved language preference from localStorage
      const savedLocale = localStorage.getItem('ntp-language');
      if (savedLocale && this.supportedLocales.includes(savedLocale)) {
        this.currentLocale = savedLocale;
      } else {
        // Detect browser language
        const browserLang = navigator.language || navigator.userLanguage || 'en';
        this.currentLocale = browserLang.startsWith('zh') ? 'zh' : 'en';
      }

      // Load messages
      await this.loadMessages(this.currentLocale);
      this.ready = true;
      document.documentElement.lang = this.currentLocale;
      // Update page after loading translations
      this.updatePage();
    })();

    try {
      await this.initPromise;
    } finally {
      this.initPromise = null;
    }
  }

  async loadMessages(locale) {
    const targetLocale = this.supportedLocales.includes(locale) ? locale : 'en';

    const cachedMessages = this.messageCache.get(targetLocale);
    if (cachedMessages) {
      this.messages = cachedMessages;
      this.currentLocale = targetLocale;
      localStorage.setItem('ntp-language', targetLocale);
      return;
    }

    try {
      const localeURL = await this.getLocaleMessagesURL(targetLocale);
      const response = await fetch(localeURL);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const messages = await response.json();
      this.messageCache.set(targetLocale, messages);
      this.messages = messages;
      this.currentLocale = targetLocale;
      // Save preference
      localStorage.setItem('ntp-language', targetLocale);
    } catch (error) {
      console.error('Failed to load messages:', error);
      // Fallback to English if translation fails
      if (targetLocale !== 'en') {
        await this.loadMessages('en');
      }
    }
  }

  async getLocaleMessagesURL(locale) {
    const manifest = await this.loadLocaleManifest();
    if (manifest && typeof manifest[locale] === 'string' && manifest[locale] !== '') {
      return manifest[locale];
    }
    return `/static/i18n/${locale}.json`;
  }

  async loadLocaleManifest() {
    if (this.localeManifest) {
      return this.localeManifest;
    }

    if (this.localeManifestPromise) {
      return this.localeManifestPromise;
    }

    this.localeManifestPromise = (async () => {
      try {
        // 清单用于定位版本化 URL，每次初始化都请求最新版本
        const response = await fetch('/static/i18n/manifest.json', { cache: 'no-cache' });
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }

        const manifest = await response.json();
        if (manifest && typeof manifest === 'object') {
          this.localeManifest = manifest;
          return this.localeManifest;
        }
      } catch (error) {
        // 本地开发或旧部署可能没有 manifest，回退到默认路径
        console.warn('i18n manifest not available, fallback to default locale files:', error);
      } finally {
        this.localeManifestPromise = null;
      }

      this.localeManifest = null;
      return null;
    })();

    return this.localeManifestPromise;
  }

  async setLocale(locale) {
    if (!this.supportedLocales.includes(locale)) {
      console.warn(`Unsupported locale: ${locale}`);
      return;
    }

    if (locale !== this.currentLocale) {
      await this.loadMessages(locale);
      // Update all translated elements
      this.updatePage();
      // Update HTML lang attribute
      document.documentElement.lang = this.currentLocale;
    }
  }

  t(key, params = {}) {
    const value = key.split('.').reduce((acc, currentKey) => acc?.[currentKey], this.messages);

    if (value === undefined) {
      if (!this.missingKeys.has(key)) {
        console.warn(`Translation not found for key: ${key}`);
        this.missingKeys.add(key);
      }
      return key;
    }

    // Replace parameters like ${param}
    if (typeof value === 'string') {
      return value.replace(/\$\{(\w+)\}/g, (match, param) => {
        // Use param value if it exists (including 0, false, empty string), otherwise keep the placeholder
        return Object.prototype.hasOwnProperty.call(params, param) ? params[param] : match;
      });
    }

    return value;
  }

  updatePage() {
    this._updateElements(document);
    // Update document title
    const titleKey = document.querySelector('title')?.getAttribute('data-i18n');
    if (titleKey) document.title = this.t(titleKey);
  }

  // Update only a specific element and its children
  updateElement(element) {
    if (!element) return;
    this._updateElements(element);
  }

  // Internal method to update all i18n attributes within a container
  _updateElements(container) {
    // Update data-i18n elements
    container.querySelectorAll('[data-i18n]').forEach(el => {
      const translation = this.t(el.dataset.i18n);
      if (el.tagName === 'INPUT' && el.hasAttribute('placeholder')) {
        el.placeholder = translation;
      } else {
        el.textContent = translation;
      }
    });

    // Update data-i18n-placeholder elements
    container.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
      el.placeholder = this.t(el.getAttribute('data-i18n-placeholder'));
    });

    // Update data-i18n-title elements
    container.querySelectorAll('[data-i18n-title]').forEach(el => {
      const translation = this.t(el.getAttribute('data-i18n-title'));
      el.title = translation;
      // Also set aria-label for better accessibility
      el.setAttribute('aria-label', translation);
    });
  }

  getCurrentLocale() {
    return this.currentLocale;
  }

  getSupportedLocales() {
    return this.supportedLocales;
  }

  getLocaleName(locale) {
    const names = {
      'en': 'English',
      'zh': '简体中文'
    };
    return names[locale] || locale;
  }

  isReady() {
    return this.ready;
  }

  // Returns the headers to include in API requests for language detection
  getRequestHeaders() {
    return {
      'X-Locale': this.currentLocale
    };
  }
}

// Create global i18n instance
const i18n = new I18n();

// Don't auto-initialize - let app.js control initialization
// app.js will call await i18n.init() when ready
