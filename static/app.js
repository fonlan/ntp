// API base path
const API = '/api';

// ===================================
// 移动端布局调整
// ===================================
// 在移动端将按钮容器移到页面底部
function adjustMobileLayout() {
    if (window.innerWidth <= 768) {
        const container = document.querySelector('.container');
        const topBar = document.querySelector('.top-bar');
        const buttonsContainer = document.querySelector('.top-right-buttons');

        if (container && topBar && buttonsContainer) {
            // 检查按钮容器是否已经被移动过
            if (buttonsContainer.parentElement !== topBar) {
                return; // 已经移动过了，不再重复移动
            }

            // 将按钮容器从 .top-bar 移到 .container 的最后
            container.appendChild(buttonsContainer);

            // 添加移动端样式类
            buttonsContainer.classList.add('mobile-bottom-buttons');
        }
    }
}

// DOM 加载完成后执行移动端布局调整
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', adjustMobileLayout);
} else {
    adjustMobileLayout();
}

// 窗口大小改变时重新检查
window.addEventListener('resize', () => {
    if (window.innerWidth <= 768) {
        adjustMobileLayout();
    } else {
        // 桌面端恢复原始位置
        const topBar = document.querySelector('.top-bar');
        const buttonsContainer = document.querySelector('.top-right-buttons');
        if (topBar && buttonsContainer && buttonsContainer.classList.contains('mobile-bottom-buttons')) {
            // 将按钮容器放回 .top-bar 的最后
            topBar.appendChild(buttonsContainer);
            buttonsContainer.classList.remove('mobile-bottom-buttons');
        }
    }
});

// Helper function to make API requests with language header
async function apiRequest(url, options = {}) {
    const headers = {
        ...i18n.getRequestHeaders(),
        ...(options.headers || {})
    };

    if (options.body && !(options.body instanceof FormData) && !headers['Content-Type']) {
        headers['Content-Type'] = 'application/json';
    }

    const response = await fetch(url, { ...options, headers });

    // 处理 401 未授权响应，重定向到登录页
    if (response.status === 401) {
        // 如果当前不在登录页面，则重定向
        if (window.location.pathname !== '/login') {
            window.location.href = '/login';
        }
        throw new Error('Unauthorized');
    }

    return response;
}

// Global state
const state = {
    bookmarks: [],
    groups: [],
    engines: [],
    currentEngine: null,
    draggedItem: null,
    containerWidth: 1400,
    bookmarkHeight: 80,
    bookmarkColumns: 0,
    mobileColumns: 2,
    cardOpacity: 95
};

// ===================================
// 滚动性能优化
// ===================================
// 使用 requestAnimationFrame 优化滚动事件
const scrollOptimizer = {
    ticking: false,
    callbacks: new Set(),

    update() {
        this.ticking = false;
        this.callbacks.forEach(callback => callback());
    },

    request(callback) {
        this.callbacks.add(callback);
        if (!this.ticking) {
            requestAnimationFrame(() => this.update());
            this.ticking = true;
        }
    },

    remove(callback) {
        this.callbacks.delete(callback);
    }
};

// 优化 passive event listeners 以提升滚动性能
document.addEventListener('touchstart', function() {}, { passive: true });
document.addEventListener('touchmove', function() {}, { passive: true });

// 防抖函数 - 用于优化滚动和 resize 事件
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// 节流函数 - 用于优化高频滚动事件
function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// Cache for frequently accessed DOM elements
const dom = {
    // Lazy getter pattern - elements are fetched on first access
    get searchInput() { return this._searchInput ||= document.getElementById('searchInput'); },
    get engineDropdown() { return this._engineDropdown ||= document.getElementById('engineDropdown'); },
    get currentEngineIcon() { return this._currentEngineIcon ||= document.getElementById('currentEngineIcon'); },
    get currentEngineName() { return this._currentEngineName ||= document.getElementById('currentEngineName'); },
    get bookmarksContainer() { return this._bookmarksContainer ||= document.getElementById('bookmarksContainer'); },
    get settingsModal() { return this._settingsModal ||= document.getElementById('settingsModal'); },
    get bookmarkModal() { return this._bookmarkModal ||= document.getElementById('bookmarkModal'); },
    get groupModal() { return this._groupModal ||= document.getElementById('groupModal'); },
    get engineEditModal() { return this._engineEditModal ||= document.getElementById('engineEditModal'); },
    get importModal() { return this._importModal ||= document.getElementById('importModal'); },
    get contextMenu() { return this._contextMenu ||= document.getElementById('contextMenu'); },
    get bookmarkGroupSelect() { return this._bookmarkGroupSelect ||= document.getElementById('bookmarkGroup'); },
    get groupsList() { return this._groupsList ||= document.getElementById('groupsList'); },
    get enginesList() { return this._enginesList ||= document.getElementById('enginesList'); },
    get groupQuickNav() { return this._groupQuickNav ||= document.getElementById('groupQuickNav'); },

    // Reset cache when DOM changes significantly
    clear() {
        this._searchInput = null;
        this._engineDropdown = null;
        this._currentEngineIcon = null;
        this._currentEngineName = null;
        this._bookmarksContainer = null;
        this._settingsModal = null;
        this._bookmarkModal = null;
        this._groupModal = null;
        this._engineEditModal = null;
        this._importModal = null;
        this._contextMenu = null;
        this._bookmarkGroupSelect = null;
        this._groupsList = null;
        this._enginesList = null;
        this._groupQuickNav = null;
    }
};

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    await i18n.init();
    loadSettings();
    initSettingsPanelNavigation();
    try {
        await Promise.all([loadSearchEngines(), loadGroups(), loadBookmarks()]);
    } catch (error) {
        console.error('Failed to load initial data:', error);
    }
    initEventListeners();
});

function initSettingsPanelNavigation() {
    const navContainer = document.querySelector('.settings-sidebar');
    if (!navContainer || navContainer._delegationInitialized) return;

    navContainer.addEventListener('click', (e) => {
        const item = e.target.closest('.settings-nav-item');
        if (!item) return;

        const panelId = item.dataset.panel;
        const navItems = document.querySelectorAll('.settings-nav-item');
        const panels = document.querySelectorAll('.settings-panel');

        navItems.forEach(nav => nav.classList.remove('active'));
        panels.forEach(panel => panel.classList.remove('active'));

        item.classList.add('active');
        const targetPanel = document.querySelector(`.settings-panel[data-panel="${panelId}"]`);
        if (targetPanel) {
            targetPanel.classList.add('active');
        }
    });

    navContainer._delegationInitialized = true;
}

function resetSettingsPanel() {
    const navItems = document.querySelectorAll('.settings-nav-item');
    const panels = document.querySelectorAll('.settings-panel');

    navItems.forEach(nav => nav.classList.remove('active'));
    panels.forEach(panel => panel.classList.remove('active'));

    const firstNav = document.querySelector('.settings-nav-item[data-panel="appearance"]');
    const firstPanel = document.querySelector('.settings-panel[data-panel="appearance"]');
    if (firstNav) firstNav.classList.add('active');
    if (firstPanel) firstPanel.classList.add('active');
}

// ===================================
// Utility Functions
// ===================================
function ensureArray(data) {
    return Array.isArray(data) ? data : [];
}

function showError(message) {
    alert(message);
}

async function loadData(url, errorMsg) {
    try {
        const res = await apiRequest(url);
        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
        const data = await res.json();
        return ensureArray(data);
    } catch (err) {
        // 如果是未授权错误，说明会话可能已过期
        // apiRequest 会处理重定向，这里只记录其他错误
        if (err.message !== 'Unauthorized') {
            console.error(errorMsg, err);
        }
        return [];
    }
}

/**
 * Initialize draggable items for sorting using event delegation
 * @param {string} itemSelector - CSS selector for draggable items
 * @param {Function} onUpdate - Callback function when order changes
 */
function initDraggable(itemSelector, onUpdate) {
    const items = document.querySelectorAll(itemSelector);
    if (items.length === 0) return;
    
    const container = items[0].parentElement;
    if (!container || container._draggableInitialized) return;

    container.addEventListener('dragstart', (e) => {
        const item = e.target.closest(itemSelector);
        if (!item) return;
        item.classList.add('dragging');
    });

    container.addEventListener('dragend', (e) => {
        const item = e.target.closest(itemSelector);
        if (!item) return;
        item.classList.remove('dragging');
        if (onUpdate) onUpdate();
    });

    container.addEventListener('dragover', (e) => {
        e.preventDefault();
        const item = e.target.closest(itemSelector);
        if (!item) return;
        
        const dragging = container.querySelector('.dragging');
        if (!dragging || dragging === item) return;

        const rect = item.getBoundingClientRect();
        const midpoint = rect.top + rect.height / 2;
        
        if (e.clientY < midpoint) {
            container.insertBefore(dragging, item);
        } else {
            container.insertBefore(dragging, item.nextSibling);
        }
    });

    container._draggableInitialized = true;
}

// ===================================
// Settings
// ===================================
/**
 * 计算合适的容器宽度（首次访问时调用）
 * 只在PC和平板视图（屏幕宽度 > 768px）时自动计算
 * 避免覆盖右上角的新建书签按钮和设置按钮
 */
function calculateOptimalContainerWidth() {
    // 移动端不进行计算，返回默认值
    if (window.innerWidth <= 768) {
        return 1400;
    }

    // 右上角按钮占用的空间
    // 按钮宽度: 44px × 2 = 88px, gap: 8px, 右侧padding: 24px
    // 预留空间: 约 150px
    const rightReservedSpace = 150;
    const containerPadding = 48; // 左右padding各24px

    // 计算可用宽度
    const availableWidth = window.innerWidth - containerPadding - rightReservedSpace;

    // 限制在合理范围内（最小800px，最大2000px）
    const minWidth = 800;
    const maxWidth = 2000;
    const optimalWidth = Math.max(minWidth, Math.min(maxWidth, availableWidth));

    // 向下取整到50的倍数，使数值更整齐
    return Math.floor(optimalWidth / 50) * 50;
}

function loadSettings() {
    // Background (color or pattern) - unified setting
    const background = localStorage.getItem('background') || '#F8FAFC';

    // Check if it's a pattern (starts with 'bg-pattern-') or a color (starts with '#')
    if (background.startsWith('bg-pattern-')) {
        // It's a pattern
        applyBackgroundPattern(background);
    } else {
        // It's a solid color
        document.body.style.background = background;
        // Clear any pattern classes
        applyBackgroundPattern('');
    }

    // Set the radio button state
    const bgRadio = document.querySelector(`input[name="background"][value="${background}"]`);
    if (bgRadio) bgRadio.checked = true;

    // Container Width - 首次访问时根据屏幕宽度自动计算（仅PC和平板）
    const savedContainerWidth = localStorage.getItem('containerWidth');
    if (savedContainerWidth === null) {
        // 首次访问，自动计算合适的容器宽度
        state.containerWidth = calculateOptimalContainerWidth();
        // 保存到 localStorage
        localStorage.setItem('containerWidth', state.containerWidth.toString());
    } else {
        state.containerWidth = parseInt(savedContainerWidth);
    }

    // Bookmark size
    state.bookmarkHeight = parseInt(localStorage.getItem('bookmarkHeight')) || 80;
    state.bookmarkColumns = parseInt(localStorage.getItem('bookmarkColumns')) || 0;
    state.mobileColumns = parseInt(localStorage.getItem('mobileColumns')) || 2;
    state.cardOpacity = parseInt(localStorage.getItem('cardOpacity')) || 95;

    // Set values to DOM elements only if they exist
    const containerWidthEl = document.getElementById('containerWidth');
    if (containerWidthEl) containerWidthEl.value = state.containerWidth;

    const bookmarkHeightEl = document.getElementById('bookmarkHeight');
    if (bookmarkHeightEl) bookmarkHeightEl.value = state.bookmarkHeight;

    const bookmarkColumnsEl = document.getElementById('bookmarkColumns');
    if (bookmarkColumnsEl) bookmarkColumnsEl.value = state.bookmarkColumns;

    const mobileColumnsEl = document.getElementById('mobileColumns');
    if (mobileColumnsEl) mobileColumnsEl.value = state.mobileColumns;

    const cardOpacityEl = document.getElementById('cardOpacity');
    if (cardOpacityEl) cardOpacityEl.value = state.cardOpacity;

    updateSizeDisplay();
    updatePreview();
    applyBookmarkSize();
    applyCardOpacity();
}

function saveSetting(key, value) {
    localStorage.setItem(key, value);
    if (key === 'background') {
        // Unified background setting (color or pattern)
        if (value.startsWith('bg-pattern-')) {
            // It's a pattern
            applyBackgroundPattern(value);
        } else {
            // It's a solid color
            document.body.style.background = value;
            // Clear any pattern classes
            applyBackgroundPattern('');
        }
    } else if (key === 'cardOpacity') {
        state.cardOpacity = parseInt(value);
        updateSizeDisplay();
        applyCardOpacity();
    } else if (key === 'containerWidth' || key === 'bookmarkColumns' || key === 'mobileColumns') {
        state[key] = parseInt(value);
        updateSizeDisplay();
        applyBookmarkSize();
    } else {
        state[key] = parseInt(value);
        updateSizeDisplay();
        updatePreview();
        applyBookmarkSize();
    }
}

function updateSizeDisplay() {
    const containerWidthValueEl = document.getElementById('containerWidthValue');
    if (containerWidthValueEl) containerWidthValueEl.textContent = state.containerWidth;

    const heightValueEl = document.getElementById('heightValue');
    if (heightValueEl) heightValueEl.textContent = state.bookmarkHeight;

    const opacityValueEl = document.getElementById('opacityValue');
    if (opacityValueEl) opacityValueEl.textContent = state.cardOpacity;

    const columnsSelect = document.getElementById('bookmarkColumns');
    const columnsValueEl = document.getElementById('columnsValue');
    if (columnsSelect && columnsValueEl && columnsSelect.selectedIndex >= 0) {
        const columnsText = columnsSelect.options[columnsSelect.selectedIndex].text;
        columnsValueEl.textContent = columnsText;
    }

    const mobileColumnsSelect = document.getElementById('mobileColumns');
    const mobileColumnsValueEl = document.getElementById('mobileColumnsValue');
    if (mobileColumnsSelect && mobileColumnsValueEl && mobileColumnsSelect.selectedIndex >= 0) {
        const mobileColumnsText = mobileColumnsSelect.options[mobileColumnsSelect.selectedIndex].text;
        mobileColumnsValueEl.textContent = mobileColumnsText;
    }
}

function updatePreview() {
    const preview = document.getElementById('bookmarkPreview');
    if (!preview) return;

    // 计算预览宽度
    let previewWidth;
    if (state.bookmarkColumns > 0) {
        previewWidth = Math.floor(state.containerWidth / state.bookmarkColumns);
    } else {
        previewWidth = 250; // 自动模式下的默认预览宽度
    }
    // Set outer dimensions of the card (including padding)
    preview.style.width = previewWidth + 'px';
    preview.style.height = state.bookmarkHeight + 'px';
    preview.style.minHeight = state.bookmarkHeight + 'px';
    
    // Set icon size to match bookmark height
    const previewIcon = preview.querySelector('.bookmark-icon');
    if (previewIcon) {
        previewIcon.style.width = state.bookmarkHeight + 'px';
        previewIcon.style.height = state.bookmarkHeight + 'px';
        previewIcon.style.minWidth = state.bookmarkHeight + 'px';
    }
}

function applyBookmarkSize() {
    const mainContainer = document.querySelector('.container');
    const groupBookmarksContainers = document.querySelectorAll('.group-bookmarks');
    const cards = document.querySelectorAll('.bookmark-card');
    const icons = document.querySelectorAll('.bookmark-icon');
    const isMobile = window.innerWidth <= 768;

    // 如果书签容器还不存在（页面刚加载时），直接返回
    // 等待 renderBookmarks() 创建元素后再调用
    if (groupBookmarksContainers.length === 0) {
        return;
    }

    // 设置主容器宽度
    if (mainContainer) {
        mainContainer.style.maxWidth = state.containerWidth + 'px';
    }

    // 计算书签宽度
    let bookmarkWidth;
    if (state.bookmarkColumns > 0) {
        bookmarkWidth = Math.floor(state.containerWidth / state.bookmarkColumns);
    } else {
        bookmarkWidth = 250; // 自动模式下的默认最小宽度
    }

    // 对每个分组容器设置列数
    groupBookmarksContainers.forEach(container => {
        if (!isMobile) {
            // 桌面端布局
            if (state.bookmarkColumns > 0) {
                container.style.gridTemplateColumns = `repeat(${state.bookmarkColumns}, 1fr)`;
            } else {
                container.style.gridTemplateColumns = `repeat(auto-fill, minmax(${bookmarkWidth}px, 1fr))`;
            }
        } else {
            // 移动端布局 - 直接设置内联样式，避免CSS变量和媒体查询的时序问题
            container.style.gridTemplateColumns = `repeat(${state.mobileColumns}, 1fr)`;
        }
    });

    // 设置每个书签卡片的高度
    cards.forEach(card => {
        card.style.width = ''; // 宽度由 grid 控制
        card.style.minHeight = state.bookmarkHeight + 'px';
        card.style.height = state.bookmarkHeight + 'px';
        card.style.overflow = 'hidden';
    });

    // 设置每个图标的尺寸与书签高度一致
    icons.forEach(icon => {
        icon.style.width = state.bookmarkHeight + 'px';
        icon.style.height = state.bookmarkHeight + 'px';
        icon.style.minWidth = state.bookmarkHeight + 'px';
    });
}

function applyCardOpacity() {
    const opacityValue = state.cardOpacity / 100;
    const rgbaColor = `rgba(255, 255, 255, ${opacityValue})`;

    // 应用到所有书签卡片
    const cards = document.querySelectorAll('.bookmark-card');
    cards.forEach(card => {
        card.style.background = rgbaColor;
    });

    // 应用到预览卡片
    const preview = document.getElementById('bookmarkPreview');
    if (preview) {
        preview.style.background = rgbaColor;
    }
}

function applyBackgroundPattern(patternClass) {
    // 移除所有已知的背景图案类
    const patternClasses = [
        'bg-pattern-dots',
        'bg-pattern-grid',
        'bg-pattern-diagonal',
        'bg-pattern-polka',
        'bg-pattern-cross',
        'bg-pattern-diamond',
        'bg-pattern-checker',
        'bg-pattern-herringbone',
        'bg-pattern-circles',
        'bg-pattern-waves',
        'bg-pattern-hexagon',
        'bg-pattern-stripes',
        'bg-pattern-dots-dark',
        'bg-pattern-grid-dark',
        'bg-pattern-diamond-dark',
        'bg-pattern-diagonal-dark',
        'bg-pattern-aurora',
        'bg-pattern-animated-gradient',
        'bg-pattern-zigzag'
    ];

    patternClasses.forEach(cls => {
        document.body.classList.remove(cls);
    });

    // 如果提供了新的图案类，则添加
    if (patternClass && patternClass.trim() !== '') {
        document.body.classList.add(patternClass);
        // 清除内联背景色，让 CSS 图案生效
        document.body.style.background = '';
    }
}

// ===================================
// Search Engine
// ===================================
async function loadSearchEngines() {
    state.engines = await loadData(`${API}/search-engines`, i18n.t('errors.loadEngineFailed'));
    state.currentEngine = state.engines.find(e => e.is_default) || state.engines[0] || null;
    renderCurrentEngine();
    renderEngineDropdown();
}

function renderCurrentEngine() {
    if (!state.currentEngine) {
        dom.currentEngineName.textContent = i18n.t('app.loading');
        return;
    }

    dom.currentEngineIcon.src = getFavicon(state.currentEngine.url, state.currentEngine.icon_path);
    dom.currentEngineName.textContent = state.currentEngine.name;
}

function renderEngineDropdown() {
    dom.engineDropdown.innerHTML = state.engines.map(e => `
        <div class="engine-dropdown-item ${state.currentEngine?.id === e.id ? 'active' : ''}"
             data-engine-id="${e.id}">
            <img src="${getFavicon(e.url, e.icon_path)}" alt="" onerror="this.onerror=null;this.src='data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22%3E%3Ctext y=%22.9em%22 font-size=%2290%22%3E🔍%3C/text%3E%3C/svg%3E'">
            <span>${escapeHtml(e.name)}</span>
        </div>
    `).join('');

    // 使用事件委托（只绑定一次到容器）
    if (!dom.engineDropdown._delegationInitialized) {
        dom.engineDropdown.addEventListener('click', (e) => {
            const item = e.target.closest('.engine-dropdown-item');
            if (!item) return;
            const engineId = parseInt(item.dataset.engineId);
            state.currentEngine = state.engines.find(eng => eng.id === engineId);
            renderCurrentEngine();
            renderEngineDropdown();
            dom.engineDropdown.classList.remove('show');
        });
        dom.engineDropdown._delegationInitialized = true;
    }
}

function getFavicon(url, iconPath) {
    // 如果有本地图标路径，直接返回本地路径
    if (iconPath && iconPath !== '') {
        return iconPath;
    }

    try {
        const domain = new URL(url).hostname;
        return `https://www.google.com/s2/favicons?domain=${domain}&sz=32`;
    } catch {
        return 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22%3E%3Ctext y=%22.9em%22 font-size=%2290%22%3E🔍%3C/text%3E%3C/svg%3E';
    }
}

function toggleEngineDropdown() {
    const dropdown = document.getElementById('engineDropdown');
    if (dropdown) {
        dropdown.classList.toggle('show');
    }
}

// ===================================
// Groups
// ===================================
async function loadGroups() {
    state.groups = await loadData(`${API}/groups`, i18n.t('errors.loadGroupFailed'));
    updateGroupSelect();
    renderSettingsGroups();
}



function updateGroupSelect() {
    dom.bookmarkGroupSelect.innerHTML = `<option value="">${i18n.t('group.ungrouped')}</option>` +
        state.groups.map(g => `<option value="${g.id}">${escapeHtml(g.name)}</option>`).join('');
}

// ===================================
// Bookmarks
// ===================================
async function loadBookmarks() {
    state.bookmarks = await loadData(`${API}/bookmarks`, i18n.t('errors.loadBookmarkFailed'));

    // 确保分组数据已加载后再渲染书签（避免竞态条件）
    // 如果分组还未加载，等待最多 100ms
    let attempts = 0;
    while (state.groups.length === 0 && attempts < 10) {
        await new Promise(resolve => setTimeout(resolve, 10));
        attempts++;
    }

    renderBookmarks();
}

function renderBookmarks() {
    if (state.bookmarks.length === 0) {
        dom.bookmarksContainer.innerHTML = `<div class="empty-state">${i18n.t('bookmark.empty')}</div>`;
        return;
    }

    // 按分组分组书签
    const groupedBookmarks = {};
    const ungroupedBookmarks = [];
    const sortedGroups = [...state.groups].sort((a, b) => a.sort_order - b.sort_order);

    sortedGroups.forEach(g => groupedBookmarks[g.id] = []);
    state.bookmarks.forEach(b => {
        if (b.group_id && groupedBookmarks[b.group_id]) {
            groupedBookmarks[b.group_id].push(b);
        } else {
            ungroupedBookmarks.push(b);
        }
    });

    // 渲染 HTML
    const html = sortedGroups
        .filter(group => groupedBookmarks[group.id]?.length > 0)
        .map(group => `
            <div class="bookmark-group" data-group-id="${group.id}">
                <h3 class="group-title" id="group-${group.id}">${escapeHtml(group.name)}</h3>
                <div class="group-bookmarks">
                    ${groupedBookmarks[group.id].map(renderBookmarkCard).join('')}
                </div>
            </div>
        `).join('');

    const ungroupedHtml = ungroupedBookmarks.length > 0 ? `
        <div class="bookmark-group" data-group-id="ungrouped">
            <h3 class="group-title" id="group-ungrouped">${i18n.t('group.ungrouped')}</h3>
            <div class="group-bookmarks">
                ${ungroupedBookmarks.map(renderBookmarkCard).join('')}
            </div>
        </div>
    ` : '';

    dom.bookmarksContainer.innerHTML = html + ungroupedHtml;
    applyBookmarkSize();
    applyCardOpacity();
    initDragAndDrop();
    initBookmarkClickHandlers();
    renderGroupQuickNav(sortedGroups, groupedBookmarks, ungroupedBookmarks.length > 0);
}

function renderGroupQuickNav(sortedGroups, groupedBookmarks, hasUngrouped) {
    const nav = dom.groupQuickNav;
    if (!nav) return;

    const visibleGroups = sortedGroups.filter(g => groupedBookmarks[g.id]?.length > 0);
    if (visibleGroups.length === 0 && !hasUngrouped) {
        nav.style.display = 'none';
        return;
    }

    let navHtml = visibleGroups.map(group => `
        <button class="group-quick-nav-item" data-target="group-${group.id}" title="${escapeHtml(group.name)}">
            <span class="group-quick-nav-dot"></span>
            <span>${escapeHtml(group.name)}</span>
        </button>
    `).join('');

    if (hasUngrouped) {
        navHtml += `
            <button class="group-quick-nav-item" data-target="group-ungrouped" title="${i18n.t('group.ungrouped')}">
                <span class="group-quick-nav-dot"></span>
                <span>${i18n.t('group.ungrouped')}</span>
            </button>
        `;
    }

    nav.innerHTML = navHtml;
    nav.style.display = '';

    initGroupQuickNavHandlers();
    initScrollTracking();
}

function initGroupQuickNavHandlers() {
    const nav = dom.groupQuickNav;
    if (!nav || nav._clickHandlerInitialized) return;

    nav.addEventListener('click', (e) => {
        const item = e.target.closest('.group-quick-nav-item');
        if (!item) return;

        const targetId = item.dataset.target;
        const targetEl = document.getElementById(targetId);
        if (targetEl) {
            targetEl.scrollIntoView({ behavior: 'smooth', block: 'start' });

            nav.querySelectorAll('.group-quick-nav-item').forEach(i => i.classList.remove('active'));
            item.classList.add('active');
        }
    });

    nav._clickHandlerInitialized = true;
}

function initScrollTracking() {
    if (window._scrollTrackingInitialized) return;

    const updateActiveGroup = throttle(() => {
        const nav = dom.groupQuickNav;
        if (!nav) return;

        const groupTitles = document.querySelectorAll('.group-title[id]');
        if (groupTitles.length === 0) return;

        let currentGroupId = null;
        const offset = 100;

        const isAtBottom = (window.innerHeight + window.scrollY) >= (document.documentElement.scrollHeight - 50);
        if (isAtBottom && groupTitles.length > 0) {
            currentGroupId = groupTitles[groupTitles.length - 1].id;
        } else {
            groupTitles.forEach(title => {
                const rect = title.getBoundingClientRect();
                if (rect.top <= offset) {
                    currentGroupId = title.id;
                }
            });

            if (!currentGroupId && groupTitles.length > 0) {
                currentGroupId = groupTitles[0].id;
            }
        }

        nav.querySelectorAll('.group-quick-nav-item').forEach(item => {
            item.classList.toggle('active', item.dataset.target === currentGroupId);
        });
    }, 100);

    window.addEventListener('scroll', updateActiveGroup, { passive: true });
    updateActiveGroup();

    window._scrollTrackingInitialized = true;
}

// 初始化书签卡片点击事件
function initBookmarkClickHandlers() {
    const cards = document.querySelectorAll('.bookmark-card');
    const menu = dom.contextMenu;

    // 使用事件委托来处理卡片事件（减少事件监听器数量）
    const bookmarksContainer = dom.bookmarksContainer;

    // 移除旧的监听器（如果存在）
    bookmarksContainer._cardClickHandler?.();
    bookmarksContainer._cardContextMenuHandler?.();
    bookmarksContainer._cardTouchStartHandler?.();
    bookmarksContainer._cardTouchEndHandler?.();
    bookmarksContainer._cardTouchMoveHandler?.();

    // 点击事件 - 打开书签
    bookmarksContainer._cardClickHandler = onEvent(bookmarksContainer, 'click', '.bookmark-card', (e, card) => {
        const url = card.dataset.url;
        const target = card.dataset.target || '_blank';
        if (url) window.open(url, target);
    });

    // 右键菜单事件
    bookmarksContainer._cardContextMenuHandler = onEvent(bookmarksContainer, 'contextmenu', '.bookmark-card', (e, card) => {
        e.preventDefault();
        showContextMenu(e, card);
    });

    // 触摸设备长按处理
    const touchState = new WeakMap();
    bookmarksContainer._cardTouchStartHandler = onEvent(bookmarksContainer, 'touchstart', '.bookmark-card', (e, card) => {
        touchState.set(card, { isLongPress: false, timer: setTimeout(() => {
            touchState.get(card).isLongPress = true;
            if (navigator.vibrate) navigator.vibrate(50);
            const touch = e.touches[0];
            showContextMenu({ preventDefault: () => {}, clientX: touch.clientX, clientY: touch.clientY, target: card }, card);
        }, 500) });
    }, { passive: true });

    bookmarksContainer._cardTouchEndHandler = onEvent(bookmarksContainer, 'touchend', '.bookmark-card', (e, card) => {
        const state = touchState.get(card);
        if (state) {
            clearTimeout(state.timer);
            if (state.isLongPress) e.preventDefault();
            touchState.delete(card);
        }
    });

    bookmarksContainer._cardTouchMoveHandler = onEvent(bookmarksContainer, 'touchmove', '.bookmark-card', (e, card) => {
        const state = touchState.get(card);
        if (state) {
            clearTimeout(state.timer);
            touchState.delete(card);
        }
    });

    // 显示右键菜单
    function showContextMenu(e, card) {
        // 根据是否是书签卡片显示不同的菜单项
        const isBookmarkCard = !!card;

        // 显示/隐藏相应的菜单项
        menu.querySelectorAll('.global-action').forEach(el => {
            el.style.display = isBookmarkCard ? 'none' : 'flex';
        });
        menu.querySelectorAll('.bookmark-action').forEach(el => {
            el.style.display = isBookmarkCard ? 'flex' : 'none';
        });
        // 分割线暂时不使用（菜单分为空白处菜单和书签菜单两种独立场景）
        menu.querySelector('.context-menu-divider').style.display = 'none';

        if (isBookmarkCard) {
            menu.dataset.currentBookmarkId = card.dataset.id;
        } else {
            menu.dataset.currentBookmarkId = null;
        }

        i18n.updateElement(menu);

        // 简单的边界检测，使用固定菜单尺寸
        const menuWidth = 180;
        const menuHeight = isBookmarkCard ? 100 : 160;
        const offset = 5;

        let x = e.clientX + offset;
        let y = e.clientY + offset;

        // 右边界检测
        if (x + menuWidth > window.innerWidth) {
            x = Math.max(5, e.clientX - menuWidth - offset);
        }

        // 下边界检测
        if (y + menuHeight > window.innerHeight) {
            y = Math.max(5, e.clientY - menuHeight - offset);
        }

        menu.style.left = x + 'px';
        menu.style.top = y + 'px';
        menu.style.display = 'block';
    }

    function hideContextMenu() {
        menu.style.display = 'none';
        menu.dataset.currentBookmarkId = null;
    }

    // 上下文菜单事件监听器（只绑定一次）
    if (!menu._menuHandlerAdded) {
        menu.addEventListener('click', (e) => {
            const item = e.target.closest('.context-menu-item');
            if (!item) return;
            const action = item.dataset.action;

            // 处理全局操作（空白处菜单）
            if (action === 'add-bookmark') {
                showBookmarkModal();
            } else if (action === 'settings') {
                // 立即显示模态框，提升响应速度
                resetSettingsPanel();
                document.getElementById('settingsModal').classList.add('show');

                // 异步加载数据和更新状态
                checkAuthStatus().then(() => {
                    try {
                        renderSettingsEngines();
                        renderSettingsGroups();
                        loadVersionInfo();
                        i18n.updatePage();
                    } catch (e) {
                        console.error('Error rendering settings content:', e);
                    }
                }).catch(e => {
                    console.error('Auth check failed:', e);
                });
            } else if (action === 'edit') {
                // 处理书签操作
                const bookmarkId = parseInt(menu.dataset.currentBookmarkId);
                editBookmark(bookmarkId);
            } else if (action === 'delete') {
                const bookmarkId = parseInt(menu.dataset.currentBookmarkId);
                deleteBookmark(bookmarkId);
            }

            hideContextMenu();
        });
        menu._menuHandlerAdded = true;
    }

    // 点击其他地方隐藏菜单（只绑定一次）
    if (!menu._docHandlerAdded) {
        document.addEventListener('click', (e) => {
            if (!menu.contains(e.target)) hideContextMenu();
        });
        menu._docHandlerAdded = true;
    }

    // 添加空白处的右键菜单（只绑定一次，避免重复绑定导致内存泄漏）
    if (!document._emptySpaceContextMenuAdded) {
        document.addEventListener('contextmenu', (e) => {
            // 只在非书签卡片和非输入框区域显示
            if (!e.target.closest('.bookmark-card') &&
                !e.target.closest('input') &&
                !e.target.closest('textarea') &&
                !e.target.closest('.modal') &&
                !e.target.closest('.context-menu')) {
                e.preventDefault();
                showContextMenu(e, null);
            }
        });
        document._emptySpaceContextMenuAdded = true;
    }

    // 添加空白处的长按菜单（移动设备）- 只绑定一次
    if (!document._emptySpaceTouchHandlersAdded) {
        const emptySpaceTouchState = { isLongPress: false, timer: null };
        
        document.addEventListener('touchstart', (e) => {
            // 只在非书签卡片区域处理
            if (e.target.closest('.bookmark-card') ||
                e.target.closest('input') ||
                e.target.closest('textarea') ||
                e.target.closest('.modal') ||
                e.target.closest('.context-menu') ||
                e.target.closest('.search-submit-btn')) {
                return;
            }

            emptySpaceTouchState.isLongPress = false;
            emptySpaceTouchState.timer = setTimeout(() => {
                emptySpaceTouchState.isLongPress = true;
                if (navigator.vibrate) navigator.vibrate(50);
                const touch = e.touches[0];
                showContextMenu({
                    preventDefault: () => {},
                    clientX: touch.clientX,
                    clientY: touch.clientY,
                    target: e.target
                }, null);
            }, 500);
        }, { passive: true });

        document.addEventListener('touchend', (e) => {
            if (emptySpaceTouchState.timer) {
                clearTimeout(emptySpaceTouchState.timer);
                emptySpaceTouchState.timer = null;
            }
            if (emptySpaceTouchState.isLongPress) {
                e.preventDefault();
            }
            emptySpaceTouchState.isLongPress = false;
        });

        document.addEventListener('touchmove', () => {
            if (emptySpaceTouchState.timer) {
                clearTimeout(emptySpaceTouchState.timer);
                emptySpaceTouchState.timer = null;
            }
            emptySpaceTouchState.isLongPress = false;
        });
        
        document._emptySpaceTouchHandlersAdded = true;
    }
}

// 事件委托辅助函数
function onEvent(container, eventType, selector, handler, options) {
    const wrappedHandler = (e) => {
        const target = e.target.closest(selector);
        if (target && container.contains(target)) {
            handler(e, target);
        }
    };
    container.addEventListener(eventType, wrappedHandler, options);
    // 返回清理函数
    return () => container.removeEventListener(eventType, wrappedHandler, options);
}

function renderBookmarkCard(b) {
    const target = b.is_new_window !== false ? '_blank' : '_self';
    const iconBgStyle = b.icon_bg_color && b.icon_bg_color !== '' 
        ? `background: ${escapeHtml(b.icon_bg_color)};` 
        : '';
    const iconSize = state.bookmarkHeight;
    const iconSizeStyle = `width: ${iconSize}px; height: ${iconSize}px; min-width: ${iconSize}px;`;
    
    // 如果有字符图标，显示字符图标；否则显示图片图标
    let iconHtml;
    if (b.icon_char && b.icon_char.trim() !== '') {
        iconHtml = `<div class="bookmark-icon bookmark-icon-char" style="${iconSizeStyle} ${iconBgStyle}" title="${escapeHtml(b.title || 'Bookmark')}">${escapeHtml(b.icon_char)}</div>`;
    } else {
        iconHtml = `<img src="${getFavicon(b.url, b.icon_path || b.icon_url)}"
                 class="bookmark-icon"
                 style="${iconSizeStyle} ${iconBgStyle}"
                 alt="${escapeHtml(b.title || 'Bookmark')} icon"
                 onerror="this.onerror=null;this.src='data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22%3E%3Ctext y=%22.9em%22 font-size=%2290%22%3E🌐%3C/text%3E%3C/svg%3E'">`;
    }
    const fallbackIcon = "data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22%3E%3Ctext y=%22.9em%22 font-size=%2290%22%3E🌐%3C/text%3E%3C/svg%3E";
    return `
        <div class="bookmark-card" draggable="true" data-id="${b.id}" data-url="${escapeHtml(b.url)}" data-target="${target}">
            ${iconHtml}
            <div class="bookmark-content">
                <div class="bookmark-title">${escapeHtml(b.title)}</div>
                ${b.description ? `<div class="bookmark-desc">${escapeHtml(b.description)}</div>` : ''}
                <div class="bookmark-url">${escapeHtml(b.url)}</div>
            </div>
        </div>
    `;
}

// ===================================
// Drag and Drop（使用事件委托，避免重复绑定）
// ===================================
function initDragAndDrop() {
    const container = dom.bookmarksContainer;
    if (!container || container._dragDelegationInitialized) return;

    container.addEventListener('dragstart', (e) => {
        const card = e.target.closest('.bookmark-card');
        if (!card) return;
        state.draggedItem = card;
        card.classList.add('dragging');
    });

    container.addEventListener('dragend', (e) => {
        const card = e.target.closest('.bookmark-card');
        if (!card) return;
        card.classList.remove('dragging');
        state.draggedItem = null;
    });

    container.addEventListener('dragover', (e) => {
        e.preventDefault();
        const card = e.target.closest('.bookmark-card');
        if (!card) return;
        
        const dragging = document.querySelector('.dragging');
        if (dragging && dragging !== card) {
            const rect = card.getBoundingClientRect();
            const midpoint = rect.y + rect.height / 2;
            card.parentNode.insertBefore(dragging, e.clientY < midpoint ? card : card.nextSibling);
        }
    });

    container.addEventListener('drop', async (e) => {
        e.preventDefault();
        const card = e.target.closest('.bookmark-card');
        if (!card) return;
        await saveBookmarkOrder();
    });

    container._dragDelegationInitialized = true;
}

async function saveBookmarkOrder() {
    const cards = document.querySelectorAll('.bookmark-card');
    const items = Array.from(cards).map((card, index) => ({
        id: parseInt(card.dataset.id),
        sort_order: index
    }));

    try {
        await apiRequest(`${API}/bookmarks/reorder`, {
            method: 'POST',
            body: JSON.stringify(items)
        });
    } catch (err) {
        console.error(i18n.t('errors.saveOrderFailed'), err);
    }
}

// ===================================
// Search
// ===================================
async function performSearch() {
    const query = document.getElementById('searchInput').value.trim();

    if (!query) {
        alert(i18n.t('search.noKeywords'));
        return;
    }

    if (!state.currentEngine) {
        alert(i18n.t('search.noEngine'));
        return;
    }

    const url = state.currentEngine.url.replace('{q}', encodeURIComponent(query));
    window.open(url, '_blank');
}

// ===================================
// Icon Tab Management
// ===================================
function switchIconTab(type) {
    // Update hidden input
    const input = document.getElementById('iconTypeInput');
    if (input) input.value = type;

    // Update tab classes
    document.querySelectorAll('.icon-tab').forEach(tab => {
        if (tab.dataset.tab === type) {
            tab.classList.add('active');
        } else {
            tab.classList.remove('active');
        }
    });

    // Update content visibility
    document.querySelectorAll('.icon-tab-content').forEach(content => {
        content.classList.remove('active');
    });
    const content = document.getElementById(`tab-content-${type}`);
    if (content) content.classList.add('active');

    // Update preview visibility
    if (type === 'image') {
        const iconUrl = document.getElementById('bookmarkIcon').value;
        const url = document.getElementById('bookmarkUrl').value;
        
        if (iconUrl) {
            showIconPreview(iconUrl);
        } else if (url) {
            showIconPreview(getFavicon(url, ''));
        } else {
            hideIconPreview();
        }
        hideCharIconPreview();
    } else {
        const char = document.getElementById('bookmarkIconChar').value.trim();
        const title = document.getElementById('bookmarkTitle').value.trim();
        
        // If char is empty but title exists, auto-fill
        if (!char && title) {
             const autoChar = title.substring(0, 2);
             document.getElementById('bookmarkIconChar').value = autoChar;
             showCharIconPreview(autoChar);
        } else if (char) {
            showCharIconPreview(char);
        } else {
            hideCharIconPreview();
        }
        hideIconPreview();
    }
}

// ===================================
// Bookmark Management
// ===================================
function showBookmarkModal(bookmark = null) {
    const modal = document.getElementById('bookmarkModal');
    const title = document.getElementById('bookmarkModalTitle');

    // 隐藏图标选择容器
    document.getElementById('iconSelectionContainer').style.display = 'none';

    if (bookmark) {
        title.textContent = i18n.t('bookmark.edit');
        document.getElementById('bookmarkId').value = bookmark.id;
        document.getElementById('bookmarkUrl').value = bookmark.url;
        document.getElementById('bookmarkTitle').value = bookmark.title;
        document.getElementById('bookmarkIcon').value = bookmark.icon_path || bookmark.icon_url || '';
        document.getElementById('bookmarkIconChar').value = bookmark.icon_char || '';
        document.getElementById('bookmarkIconBgColor').value = bookmark.icon_bg_color || '';
        if (bookmark.icon_bg_color && /^#[0-9A-Fa-f]{6}$/.test(bookmark.icon_bg_color)) {
            document.getElementById('bookmarkIconBgColorPicker').value = bookmark.icon_bg_color;
        } else {
            document.getElementById('bookmarkIconBgColorPicker').value = '#f1f5f9';
        }
        document.getElementById('bookmarkGroup').value = bookmark.group_id !== null && bookmark.group_id !== undefined ? bookmark.group_id : '';
        document.getElementById('bookmarkDesc').value = bookmark.description || '';
        document.getElementById('bookmarkNewWindow').checked = bookmark.is_new_window !== false; // 默认true

        // 根据是否有字符图标设置图标类型
        if (bookmark.icon_char && bookmark.icon_char.trim() !== '') {
            // 字符图标模式
            switchIconTab('char');
        } else {
            // 图片图标模式
            switchIconTab('image');
            // Show icon preview - use getFavicon to get correct icon
            const faviconUrl = getFavicon(bookmark.url, bookmark.icon_path || bookmark.icon_url);
            showIconPreview(faviconUrl);
        }
        updateIconBgColorPreview(bookmark.icon_bg_color || '');
    } else {
        title.textContent = i18n.t('bookmark.add');
        // 先设置默认值，再重置表单
        document.getElementById('bookmarkNewWindow').checked = true;
        document.getElementById('bookmarkForm').reset();
        document.getElementById('bookmarkId').value = '';
        document.getElementById('uploadFileName').textContent = '';
        document.getElementById('iconUpload').value = '';
        document.getElementById('bookmarkIconBgColor').value = '';
        document.getElementById('bookmarkIconBgColorPicker').value = '#f1f5f9';
        
        // reset() 会重置 checkbox 和 radio，但需要重置 tab
        document.getElementById('bookmarkNewWindow').checked = true;
        switchIconTab('image');
        hideIconPreview();
        hideCharIconPreview();
        updateIconBgColorPreview('');
    }

    modal.classList.add('show');
}

function showIconPreview(iconUrl) {
    const preview = document.getElementById('iconPreview');
    if (!preview) return;
    preview.src = iconUrl;
    preview.onerror = () => {
        preview.onerror = null;
        preview.src = 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22%3E%3Ctext y=%22.9em%22 font-size=%2290%22%3E🌐%3C/text%3E%3C/svg%3E';
    };
    preview.style.display = 'block';
}

function hideIconPreview() {
    const preview = document.getElementById('iconPreview');
    if (preview) preview.style.display = 'none';
}

function showCharIconPreview(char) {
    const preview = document.getElementById('iconCharPreviewBox');
    if (!preview) return;
    preview.textContent = char || 'AB';
    preview.style.display = 'flex';
}

function hideCharIconPreview() {
    const preview = document.getElementById('iconCharPreviewBox');
    if (preview) preview.style.display = 'none';
}

function updateIconBgColorPreview(color) {
    const preview = document.getElementById('iconPreview');
    const charPreview = document.getElementById('iconCharPreviewBox');
    const bgColor = color && color !== '' ? color : 'var(--bg-hover)';
    
    if (preview) {
        preview.style.background = bgColor;
    }
    if (charPreview) {
        charPreview.style.background = bgColor;
    }
}

async function fetchIcon() {
    const url = document.getElementById('bookmarkUrl').value.trim();
    if (!url) {
        alert(i18n.t('bookmark.enterUrlFirst'));
        return;
    }

    try {
        // 调用新的 API 获取网站元数据（包括所有图标）
        const response = await apiRequest(`${API}/fetch-metadata`, {
            method: 'POST',
            body: JSON.stringify({ url })
        });

        if (!response.ok) {
            // 尝试读取服务器返回的错误信息
            let errorMsg = 'Failed to fetch metadata';
            try {
                const errorData = await response.json();
                if (errorData.error) {
                    errorMsg = errorData.error;
                }
            } catch (e) {
                // 如果无法解析 JSON，使用状态码作为错误信息
                errorMsg = `HTTP ${response.status}: ${response.statusText}`;
            }
            throw new Error(errorMsg);
        }

        const data = await response.json();

        // 如果有图标选项，显示选择界面
        if (data.icon_options && data.icon_options.length > 0) {
            showIconSelection(data.icon_options, url);

            // 自动填充标题（仅当标题栏为空时）
            if (data.title) {
                const titleInput = document.getElementById('bookmarkTitle');
                if (!titleInput.value.trim()) {
                    titleInput.value = data.title;
                }
            }
        }
    } catch (err) {
        console.error('Failed to fetch icons:', err);
        alert(i18n.t('fetch.fetchFailed') + ': ' + err.message);
    }
}

// 图标文件扩展名映射
const ICON_EXT_MAP = { '.png': 'PNG', '.jpg': 'JPG', '.jpeg': 'JPG', '.gif': 'GIF', '.svg': 'SVG', '.ico': 'ICO', '.webp': 'WEBP', '.bmp': 'BMP' };

// 格式化图标信息（尺寸和类型）
function formatIconInfo(icon) {
    // 尺寸信息
    let sizeInfo = '';
    if (icon.sizes) sizeInfo = icon.sizes.split(' ').map(s => s.replace('x', '×')).join(', ');
    else if (icon.is_favicon) sizeInfo = 'favicon.ico';
    else sizeInfo = i18n.t('icon.unknownSize');

    // 类型信息 - 从 URL 后缀或 MIME type
    let typeInfo = '';
    if (icon.url) {
        try {
            const ext = '.' + new URL(icon.url).pathname.toLowerCase().split('.').pop();
            typeInfo = ICON_EXT_MAP[ext] || '';
        } catch { }
    }
    if (!typeInfo && icon.type) {
        typeInfo = icon.type.split('/')?.[1]?.toUpperCase() || '';
    }

    return { sizeInfo, typeInfo };
}

// 生成图标选项 HTML
function createIconOptionHtml(icon, index) {
    const { sizeInfo, typeInfo } = formatIconInfo(icon);
    const escapedUrl = escapeHtml(icon.url);

    return `
        <div class="icon-option" data-icon-url="${escapedUrl}" data-index="${index}" title="${escapedUrl}">
            <img src="${escapedUrl}" alt="Icon ${index + 1}"
                 onerror="this.onerror=null;this.src='data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22%3E%3Ctext y=%22.9em%22 font-size=%2290%22%3E❌%3C/text%3E%3C/svg%3E'">
            <div class="icon-option-info">
                <div class="icon-option-size">${sizeInfo}</div>
                ${typeInfo ? `<div class="icon-option-type">${typeInfo}</div>` : ''}
            </div>
        </div>
    `;
}

function showIconSelection(iconOptions, websiteUrl) {
    const container = document.getElementById('iconSelectionContainer');
    const list = document.getElementById('iconSelectionList');

    list.innerHTML = iconOptions.map((icon, index) => createIconOptionHtml(icon, index)).join('');
    container.style.display = 'block';

    // 使用事件委托处理图标点击（只绑定一次）
    if (!list._iconSelectionHandlerAdded) {
        list.addEventListener('click', (e) => {
            const option = e.target.closest('.icon-option');
            if (!option) return;

            list.querySelectorAll('.icon-option').forEach(o => o.classList.remove('selected'));
            option.classList.add('selected');
            selectIcon(option.dataset.iconUrl);
        });
        list._iconSelectionHandlerAdded = true;
    }

    // 绑定"使用 Google 图标"按钮事件（防止重复绑定）
    const useGoogleBtn = document.getElementById('useGoogleIconBtn');
    if (useGoogleBtn) {
        // 使用 onclick 覆盖旧处理程序，或者是新的事件监听器
        useGoogleBtn.onclick = () => selectIcon(getFavicon(websiteUrl));
    }
}

// 选择图标并隐藏选择界面
function selectIcon(iconUrl) {
    document.getElementById('bookmarkIcon').value = iconUrl;
    showIconPreview(iconUrl);
    document.getElementById('iconSelectionContainer').style.display = 'none';
}

async function uploadIcon(file) {
    if (!file) return;

    // Validate file size (max 512KB)
    if (file.size > 512 * 1024) {
        alert(i18n.t('upload.fileSizeLimit'));
        return;
    }

    // Validate file type
    const allowedTypes = ['image/png', 'image/jpeg', 'image/gif', 'image/svg+xml', 'image/x-icon'];
    if (!allowedTypes.includes(file.type)) {
        alert(i18n.t('upload.unsupportedFormat'));
        return;
    }

    const formData = new FormData();
    formData.append('icon', file);

    try {
        const response = await apiRequest(`${API}/upload-icon`, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error(i18n.t('upload.uploadFailed'));
        }

        const result = await response.json();

        // Use local icon path
        document.getElementById('bookmarkIcon').value = result.icon_path;
        showIconPreview(result.icon_path);

        // Show file name
        document.getElementById('uploadFileName').textContent = file.name;
    } catch (err) {
        console.error(i18n.t('upload.uploadFailed'), err);
        alert(i18n.t('upload.uploadFailed'));
    }
}

async function editBookmark(id) {
    const bookmark = state.bookmarks.find(b => b.id === id);
    if (bookmark) showBookmarkModal(bookmark);
}

async function deleteBookmark(id) {
    if (!confirm(i18n.t('bookmark.deleteConfirm'))) return;

    try {
        await apiRequest(`${API}/bookmark/${id}`, { method: 'DELETE' });
        await Promise.all([loadBookmarks(), loadGroups()]);
    } catch (err) {
        console.error(i18n.t('errors.deleteFailed'), err);
        showError(i18n.t('errors.deleteFailed'));
    }
}

// ===================================
// Group Management
// ===================================
function showGroupModal(group = null) {
    const modal = document.getElementById('groupModal');
    document.getElementById('groupModalTitle').textContent = i18n.t(group ? 'group.edit' : 'group.add');

    if (group) {
        document.getElementById('groupId').value = group.id;
        document.getElementById('groupName').value = group.name;
    } else {
        document.getElementById('groupForm').reset();
        document.getElementById('groupId').value = '';
    }

    modal.classList.add('show');
}

async function editGroup(id) {
    const group = state.groups.find(g => g.id === id);
    if (group) showGroupModal(group);
}

async function deleteGroup(id) {
    if (!confirm(i18n.t('group.deleteConfirm'))) return;

    try {
        await apiRequest(`${API}/groups/${id}`, { method: 'DELETE' });
        await loadGroups();
        renderSettingsGroups();
    } catch (err) {
        console.error(i18n.t('errors.deleteFailed'), err);
        showError(i18n.t('errors.deleteFailed'));
    }
}

function renderSettingsGroups() {
    const container = document.getElementById('groupsList');
    container.innerHTML = state.groups.map(g => `
        <div class="settings-item" draggable="true" data-id="${g.id}">
            <div class="settings-item-info">
                <div class="settings-item-name">${escapeHtml(g.name)}</div>
                <div class="settings-item-url">${g.bookmark_count} ${i18n.t('group.bookmarks')}</div>
            </div>
            <div class="settings-item-actions">
                <button class="btn-icon" data-action="edit-group" data-id="${g.id}" title="${i18n.t('actions.edit')}">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                </button>
                <button class="btn-icon btn-icon-danger" data-action="delete-group" data-id="${g.id}" title="${i18n.t('actions.delete')}">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/>
                        <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                    </svg>
                </button>
            </div>
        </div>
    `).join('');

    initGroupDragAndDrop();
}

function initGroupListActions() {
    const container = document.getElementById('groupsList');
    if (!container) return;
    
    // 只有当没有初始化过才绑定
    if (container._actionsInitialized) return;
    container._actionsInitialized = true;

    container.addEventListener('click', async (e) => {
        const btn = e.target.closest('[data-action]');
        if (!btn) return;

        const action = btn.dataset.action;
        const id = parseInt(btn.dataset.id);

        if (action === 'edit-group') {
            const group = state.groups.find(g => g.id === id);
            if (group) showGroupModal(group);
        } else if (action === 'delete-group') {
            if (!confirm(i18n.t('group.deleteConfirm'))) return;
            try {
                await apiRequest(`${API}/groups/${id}`, { method: 'DELETE' });
                await loadGroups();
                renderSettingsGroups();
            } catch (err) {
                console.error(i18n.t('errors.deleteFailed'), err);
                showError(i18n.t('errors.deleteFailed'));
            }
        }
    });
}

// 分组拖拽排序
function initGroupDragAndDrop() {
    initDraggable('#groupsList .settings-item', async () => {
        const items = document.querySelectorAll('#groupsList .settings-item');
        const groupItems = Array.from(items).map((item, index) => ({
            id: parseInt(item.dataset.id),
            sort_order: index
        }));

        try {
            await apiRequest(`${API}/groups/reorder`, {
                method: 'POST',
                body: JSON.stringify(groupItems)
            });
            await loadGroups();
        } catch (err) {
            console.error(i18n.t('errors.saveOrderFailed'), err);
            showError(i18n.t('errors.saveOrderFailed'));
        }
    });
}

// ===================================
// Search Engine Management
// ===================================
function showEngineModal(engine = null) {
    const modal = document.getElementById('engineEditModal');
    document.getElementById('engineEditModalTitle').textContent = i18n.t(engine ? 'searchEngine.edit' : 'searchEngine.add');

    if (engine) {
        document.getElementById('engineId').value = engine.id;
        document.getElementById('engineName').value = engine.name;
        document.getElementById('engineUrl').value = engine.url;
        document.getElementById('enginePlaceholder').value = engine.placeholder || '';
        document.getElementById('engineDefault').checked = engine.is_default;
    } else {
        document.getElementById('engineEditForm').reset();
        document.getElementById('engineId').value = '';
    }

    modal.classList.add('show');
}

async function editEngine(id) {
    const engine = state.engines.find(e => e.id === id);
    if (engine) showEngineModal(engine);
}

async function deleteEngine(id) {
    if (!confirm(i18n.t('searchEngine.deleteConfirm'))) return;

    try {
        await apiRequest(`${API}/search-engine/${id}`, { method: 'DELETE' });
        await loadSearchEngines();
        renderSettingsEngines();
    } catch (err) {
        console.error(i18n.t('errors.deleteFailed'), err);
        showError(i18n.t('errors.deleteFailed'));
    }
}

function renderSettingsEngines() {
    const container = document.getElementById('enginesList');
    container.innerHTML = state.engines.map(e => `
        <div class="settings-item">
            <div class="settings-item-info">
                <div class="settings-item-name">
                    ${escapeHtml(e.name)}
                    ${e.is_default ? `<small>(${i18n.t('searchEngine.default')})</small>` : ''}
                </div>
                <div class="settings-item-url">${escapeHtml(e.url)}</div>
            </div>
            <div class="settings-item-actions">
                <button class="btn-icon" data-action="edit-engine" data-id="${e.id}" title="${i18n.t('actions.edit')}">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                </button>
                ${!e.is_default ? `<button class="btn-icon btn-icon-danger" data-action="delete-engine" data-id="${e.id}" title="${i18n.t('actions.delete')}">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/>
                        <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                    </svg>
                </button>` : ''}
            </div>
        </div>
    `).join('');
}

function initEngineListActions() {
    const container = document.getElementById('enginesList');
    if (!container) return;

    // 只有当没有初始化过才绑定
    if (container._actionsInitialized) return;
    container._actionsInitialized = true;

    container.addEventListener('click', async (e) => {
        const btn = e.target.closest('[data-action]');
        if (!btn) return;

        const action = btn.dataset.action;
        const id = parseInt(btn.dataset.id);

        if (action === 'edit-engine') {
            const engine = state.engines.find(eng => eng.id === id);
            if (engine) showEngineModal(engine);
        } else if (action === 'delete-engine') {
            if (!confirm(i18n.t('searchEngine.deleteConfirm'))) return;
            try {
                await apiRequest(`${API}/search-engine/${id}`, { method: 'DELETE' });
                await loadSearchEngines();
                renderSettingsEngines();
            } catch (err) {
                console.error(i18n.t('errors.deleteFailed'), err);
                showError(i18n.t('errors.deleteFailed'));
            }
        }
    });
}

// ===================================
// Import/Export
// ===================================
function showImportModal() {
    const modal = document.getElementById('importModal');
    document.getElementById('importFile').value = '';
    document.getElementById('importResult').className = 'import-result';
    document.getElementById('importResult').style.display = 'none';
    // 重置导入模式为追加模式
    document.querySelector('input[name="importMode"][value="append"]').checked = true;
    modal.classList.add('show');
}

async function importBookmarks() {
    const fileInput = document.getElementById('importFile');
    const result = document.getElementById('importResult');

    if (!fileInput.files.length) {
        alert(i18n.t('errors.noFileSelected'));
        return;
    }

    // 获取选择的导入模式
    const mode = document.querySelector('input[name="importMode"]:checked').value;

    const formData = new FormData();
    formData.append('file', fileInput.files[0]);
    formData.append('mode', mode);

    try {
        const res = await apiRequest(`${API}/bookmarks/import`, {
            method: 'POST',
            body: formData
        });
        const data = await res.json();

        result.className = 'import-result success';
        result.innerHTML = i18n.t('import.success', { imported: data.imported, failed: data.failed });
        result.style.display = 'block';

        if (data.imported > 0) {
            // 先加载分组数据，再加载书签数据并渲染
            await loadGroups();
            await loadBookmarks();
        }
    } catch (err) {
        result.className = 'import-result error';
        result.textContent = i18n.t('import.failed') + err.message;
        result.style.display = 'block';
    }
}

async function exportBookmarks() {
    try {
        const res = await apiRequest(`${API}/bookmarks/export`);
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `bookmarks_${new Date().toISOString().slice(0, 10)}.html`;
        a.click();
        window.URL.revokeObjectURL(url);
    } catch (err) {
        console.error(i18n.t('export.failed'), err);
        alert(i18n.t('export.failed'));
    }
}

// ===================================
// Authentication
// ===================================

// Check authentication status
async function checkAuthStatus() {
    try {
        const response = await apiRequest('/api/auth/check');
        const data = await response.json();

        // Show account section only if authentication is enabled and user is authenticated
        const accountSection = document.getElementById('accountSection');
        if (accountSection) {
            const shouldShow = data.enabled && data.authenticated;
            accountSection.style.display = shouldShow ? 'block' : 'none';
        }
    } catch (error) {
        console.error('Failed to check auth status:', error);
        // Hide account section on error
        const accountSection = document.getElementById('accountSection');
        if (accountSection) {
            accountSection.style.display = 'none';
        }
    }
}

// Handle logout
async function handleLogout() {
    if (!confirm(i18n.t('settings.logoutConfirm'))) {
        return;
    }

    try {
        const response = await apiRequest('/api/logout', {
            method: 'POST'
        });

        const data = await response.json();

        if (data.success) {
            // Redirect to login page
            window.location.href = '/login.html';
        } else {
            alert(i18n.t('settings.logoutFailed'));
        }
    } catch (error) {
        console.error('Logout error:', error);
        alert(i18n.t('settings.logoutFailed'));
    }
}

async function loadVersionInfo() {
    try {
        const response = await fetch('/api/version');
        const data = await response.json();
        const versionEl = document.getElementById('appVersion');
        if (versionEl && data.version) {
            versionEl.textContent = data.version;
        }
    } catch (error) {
        console.error('Failed to load version:', error);
    }
}

// ===================================
// Event Listeners Initialization
// ===================================
function initEventListeners() {
    // Search engine selector
    document.getElementById('searchEngineSelector').addEventListener('click', (e) => {
        if (!e.target.closest('.engine-dropdown-item')) {
            toggleEngineDropdown();
        }
    });

    // Click outside to close dropdown
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.search-container')) {
            document.getElementById('engineDropdown').classList.remove('show');
        }
    });

    // Search button and input field
    document.getElementById('searchBtn').addEventListener('click', performSearch);
    document.getElementById('searchInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });

    // Unified background selection (colors and patterns)
    document.querySelectorAll('input[name="background"]').forEach(radio => {
        radio.addEventListener('change', (e) => {
            saveSetting('background', e.target.value);
        });
    });

    // Bookmark size controls
    document.getElementById('containerWidth').addEventListener('input', (e) => {
        saveSetting('containerWidth', e.target.value);
    });

    document.getElementById('bookmarkHeight').addEventListener('input', (e) => {
        saveSetting('bookmarkHeight', e.target.value);
    });

    document.getElementById('bookmarkColumns').addEventListener('change', (e) => {
        saveSetting('bookmarkColumns', e.target.value);
    });

    document.getElementById('mobileColumns').addEventListener('change', (e) => {
        saveSetting('mobileColumns', e.target.value);
    });

    document.getElementById('cardOpacity').addEventListener('input', (e) => {
        saveSetting('cardOpacity', e.target.value);
    });

    // Modal close
    document.querySelectorAll('.close, .close-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            btn.closest('.modal').classList.remove('show');
        });
    });

    // Click outside modal to close
    // 记录鼠标按下时的位置，确保点击和释放在同一位置才关闭弹框
    let mouseDownTarget = null;
    document.addEventListener('mousedown', (e) => {
        mouseDownTarget = e.target;
    });

    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', (e) => {
            // 只有当鼠标按下和释放在同一个元素上，且点击的是遮罩层时才关闭
            if (e.target === modal && mouseDownTarget === modal) {
                modal.classList.remove('show');
            }
        });
    });

    // Group management button
    document.getElementById('addGroupBtn').addEventListener('click', () => showGroupModal());

    // Search engine management button
    document.getElementById('addEngineBtn').addEventListener('click', () => showEngineModal());

    // Language selection
    const languageSelect = document.getElementById('languageSelect');
    if (languageSelect) {
        // Set initial value
        languageSelect.value = i18n.getCurrentLocale();

        languageSelect.addEventListener('change', async (e) => {
            await i18n.setLocale(e.target.value);
            renderGroups();
            renderBookmarks();
            updateGroupSelect();
            renderCurrentEngine();
            renderEngineDropdown();
        });
    }

    // Fetch icon button
    document.getElementById('fetchIconBtn').addEventListener('click', fetchIcon);

    // Update preview when icon URL changes
    document.getElementById('bookmarkIcon').addEventListener('input', (e) => {
        const iconUrl = e.target.value.trim();
        if (iconUrl) {
            showIconPreview(iconUrl);
        } else {
            hideIconPreview();
        }
    });

    // Icon upload button click
    document.getElementById('uploadIconBtn').addEventListener('click', () => {
        const iconUpload = document.getElementById('iconUpload');
        iconUpload.value = '';
        iconUpload.click();
    });

    // Icon upload
    document.getElementById('iconUpload').addEventListener('change', (e) => {
        const file = e.target.files[0];
        if (file) {
            uploadIcon(file);
        }
        // 清空 file input 的 value，确保下次选择相同文件时也能触发 change 事件
        e.target.value = '';
    });

    // Icon type switching (Tab)
    document.querySelectorAll('.icon-tab').forEach(tab => {
        tab.addEventListener('click', (e) => {
            switchIconTab(e.target.dataset.tab);
        });
    });

    // Character icon input - update preview
    document.getElementById('bookmarkIconChar').addEventListener('input', (e) => {
        const char = e.target.value.trim();
        if (char) {
            showCharIconPreview(char);
        } else {
            hideCharIconPreview();
        }
    });

    // Icon background color picker
    document.getElementById('bookmarkIconBgColorPicker').addEventListener('input', (e) => {
        document.getElementById('bookmarkIconBgColor').value = e.target.value;
        updateIconBgColorPreview(e.target.value);
    });

    document.getElementById('bookmarkIconBgColor').addEventListener('input', (e) => {
        const color = e.target.value.trim();
        if (color && color !== 'transparent' && /^#[0-9A-Fa-f]{6}$/.test(color)) {
            document.getElementById('bookmarkIconBgColorPicker').value = color;
        }
        updateIconBgColorPreview(color);
    });

    document.getElementById('clearIconBgColorBtn').addEventListener('click', () => {
        document.getElementById('bookmarkIconBgColor').value = '';
        document.getElementById('bookmarkIconBgColorPicker').value = '#f1f5f9';
        updateIconBgColorPreview('');
    });

    // Import/Export buttons
    document.getElementById('importBtn').addEventListener('click', showImportModal);
    document.getElementById('importSubmitBtn').addEventListener('click', importBookmarks);
    document.getElementById('exportBtn').addEventListener('click', exportBookmarks);

    // Logout button
    document.getElementById('logoutBtn').addEventListener('click', handleLogout);

    // Bookmark form submit
    document.getElementById('bookmarkForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        // 获取图标类型
        const iconType = document.getElementById('iconTypeInput').value;
        let iconChar = '';
        let iconUrl = document.getElementById('bookmarkIcon').value;
        let iconBgColor = document.getElementById('bookmarkIconBgColor').value.trim();

        // 如果选择字符图标
        if (iconType === 'char') {
            const charInput = document.getElementById('bookmarkIconChar').value.trim();
            // 如果用户没有输入，使用标题的前2个字符
            iconChar = charInput || document.getElementById('bookmarkTitle').value.substring(0, 2);
            iconUrl = ''; // 清空图片 URL
        }

        await saveForm('bookmark', {
            url: document.getElementById('bookmarkUrl').value,
            title: document.getElementById('bookmarkTitle').value,
            icon_url: iconUrl,
            icon_char: iconChar,
            icon_bg_color: iconBgColor,
            group_id: parseGroupId(document.getElementById('bookmarkGroup').value),
            description: document.getElementById('bookmarkDesc').value,
            is_new_window: document.getElementById('bookmarkNewWindow').checked
        }, async (result, existingId) => {
            // existingId 有值表示编辑，空表示新增
            if (existingId) {
                // 编辑模式：更新本地状态中的书签
                const index = state.bookmarks.findIndex(b => b.id === parseInt(existingId));
                if (index !== -1) {
                    state.bookmarks[index] = result;
                }
            } else {
                // 新增模式：将新书签添加到本地状态
                state.bookmarks.push(result);
            }
            // 重新渲染书签列表
            renderBookmarks();
            // 更新分组下拉选择（因为分组可能变化）
            updateGroupSelect();
        });
    });

    // Group form submit
    document.getElementById('groupForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        await saveForm('group', {
            name: document.getElementById('groupName').value
        }, () => loadGroups());
    });

    // Search engine form submit
    document.getElementById('engineEditForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        await saveForm('search-engine', {
            name: document.getElementById('engineName').value,
            url: document.getElementById('engineUrl').value,
            placeholder: document.getElementById('enginePlaceholder').value,
            is_default: document.getElementById('engineDefault').checked
        }, async () => { await loadSearchEngines(); renderSettingsEngines(); });
    });

    // Window resize - 重新应用书签尺寸设置以适配桌面/移动端切换
    let resizeTimer;
    window.addEventListener('resize', () => {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(() => {
            applyBookmarkSize();
        }, 100);
    });

    // 初始化设置列表的操作按钮事件（委托）
    initGroupListActions();
    initEngineListActions();
}

// HTML escape
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 统一的表单保存函数
async function saveForm(resource, data, afterSave) {
    const idFieldMap = { 'bookmark': 'bookmarkId', 'group': 'groupId', 'search-engine': 'engineId' };
    const id = document.getElementById(idFieldMap[resource]).value;
    const resourcePathMap = {
        'bookmark': { list: 'bookmarks', single: 'bookmark' },
        'group': { list: 'groups', single: 'groups' },
        'search-engine': { list: 'search-engines', single: 'search-engine' }
    };
    const paths = resourcePathMap[resource];
    const url = id ? `${API}/${paths.single}/${id}` : `${API}/${paths.list}`;
    const method = id ? 'PUT' : 'POST';
    const modalMap = { 'bookmark': 'bookmarkModal', 'group': 'groupModal', 'search-engine': 'engineEditModal' };
    const modal = document.getElementById(modalMap[resource]);

    try {
        const response = await apiRequest(url, { method, body: JSON.stringify(data) });
        const result = await response.json();
        modal.classList.remove('show');
        await afterSave(result, id);
    } catch (err) {
        console.error(i18n.t('errors.saveFailed'), err);
        alert(i18n.t('errors.saveFailed'));
    }
}

// 解析分组ID，处理空值
function parseGroupId(value) {
    return value === '' || value == null ? null : parseInt(value, 10);
}
