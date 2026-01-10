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
      return value.replace(/\$\{(\w+)\}/g, (match, param) => params[param] || match);
    }

    return value;
  }

  updatePage() {
    // Update all elements with data-i18n attribute
    document.querySelectorAll('[data-i18n]').forEach(element => {
      const key = element.getAttribute('data-i18n');
      const translation = this.t(key);

      if (element.tagName === 'INPUT' && element.hasAttribute('placeholder')) {
        element.placeholder = translation;
      } else {
        element.textContent = translation;
      }
    });

    // Update all elements with data-i18n-placeholder attribute
    document.querySelectorAll('[data-i18n-placeholder]').forEach(element => {
      const key = element.getAttribute('data-i18n-placeholder');
      element.placeholder = this.t(key);
    });

    // Update all elements with data-i18n-title attribute
    document.querySelectorAll('[data-i18n-title]').forEach(element => {
      const key = element.getAttribute('data-i18n-title');
      element.title = this.t(key);
    });

    // Update document title
    const titleKey = document.querySelector('title').getAttribute('data-i18n');
    if (titleKey) {
      document.title = this.t(titleKey);
    }
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
