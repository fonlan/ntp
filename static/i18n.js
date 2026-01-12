// i18n internationalization utility
class I18n {
  constructor() {
    this.currentLocale = 'en';
    this.messages = {};
    this.supportedLocales = ['en', 'zh'];
    this.ready = false;
  }

  async init() {
    // Avoid duplicate initialization
    if (this.ready) {
      return;
    }

    // Get saved language preference from localStorage
    const savedLocale = localStorage.getItem('ntp-language');
    if (savedLocale && this.supportedLocales.includes(savedLocale)) {
      this.currentLocale = savedLocale;
    } else {
      // Detect browser language
      const browserLang = navigator.language || navigator.userLanguage;
      if (browserLang.startsWith('zh')) {
        this.currentLocale = 'zh';
      } else {
        this.currentLocale = 'en';
      }
    }

    // Load messages
    await this.loadMessages(this.currentLocale);
    this.ready = true;
    // Update page after loading translations
    this.updatePage();
  }

  async loadMessages(locale) {
    try {
      const response = await fetch(`/static/i18n/${locale}.json`);
      this.messages = await response.json();
      this.currentLocale = locale;
      // Save preference
      localStorage.setItem('ntp-language', locale);
    } catch (error) {
      console.error('Failed to load messages:', error);
      // Fallback to English if translation fails
      if (locale !== 'en') {
        await this.loadMessages('en');
      }
    }
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
      document.documentElement.lang = locale;
    }
  }

  t(key, params = {}) {
    const keys = key.split('.');
    let value = this.messages;

    for (const k of keys) {
      if (value && value[k] !== undefined) {
        value = value[k];
      } else {
        console.warn(`Translation not found for key: ${key}`);
        return key;
      }
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
      const translation = this.t(el.getAttribute('data-i18n'));
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
