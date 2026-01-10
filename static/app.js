// API base path
const API = '/api';

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
let state = {
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

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    await i18n.init();
    loadSettings();
    try {
        await Promise.all([loadSearchEngines(), loadGroups(), loadBookmarks()]);
    } catch (error) {
        console.error('Failed to load initial data:', error);
        // 如果是因为未授权导致的错误，会在 apiRequest 中处理重定向
        // 这里只记录其他类型的错误
    }
    initEventListeners();
});

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

// ===================================
// Settings
// ===================================
function loadSettings() {
    // Background color
    const bgColor = localStorage.getItem('bgColor') || '#F8FAFC';
    document.body.style.background = bgColor;
    const bgColorRadio = document.querySelector(`input[name="bgColor"][value="${bgColor}"]`);
    if (bgColorRadio) bgColorRadio.checked = true;

    // Bookmark size
    state.containerWidth = parseInt(localStorage.getItem('containerWidth')) || 1400;
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
    if (key === 'bgColor') {
        document.body.style.background = value;
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
}

function applyBookmarkSize() {
    const mainContainer = document.querySelector('.container');
    const groupBookmarksContainers = document.querySelectorAll('.group-bookmarks');
    const cards = document.querySelectorAll('.bookmark-card');
    const isMobile = window.innerWidth <= 768;

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
        // 桌面端布局 - 只在桌面端设置内联样式
        if (!isMobile) {
            if (state.bookmarkColumns > 0) {
                container.style.gridTemplateColumns = `repeat(${state.bookmarkColumns}, 1fr)`;
            } else {
                container.style.gridTemplateColumns = `repeat(auto-fill, minmax(${bookmarkWidth}px, 1fr))`;
            }
        } else {
            // 移动端清除内联样式，让 CSS 变量生效
            container.style.gridTemplateColumns = '';
        }

        // 移动端布局（通过 CSS 变量传递）
        container.style.setProperty('--mobile-columns', state.mobileColumns);

        // 桌面端布局也通过 CSS 变量传递（可选，用于一致性）
        if (state.bookmarkColumns > 0) {
            container.style.setProperty('--desktop-columns', state.bookmarkColumns);
            container.style.setProperty('--desktop-width', bookmarkWidth + 'px');
        } else {
            container.style.setProperty('--desktop-columns', 'auto');
            container.style.setProperty('--desktop-width', bookmarkWidth + 'px');
        }
    });

    // 设置每个书签卡片的高度
    cards.forEach(card => {
        card.style.width = ''; // 宽度由 grid 控制
        card.style.minHeight = state.bookmarkHeight + 'px';
        card.style.height = state.bookmarkHeight + 'px';
        card.style.overflow = 'hidden';
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
    const iconEl = document.getElementById('currentEngineIcon');
    const nameEl = document.getElementById('currentEngineName');

    if (!iconEl || !nameEl) return;

    if (state.currentEngine) {
        // 优先使用本地缓存图标，否则使用 Google favicon
        iconEl.src = getFavicon(state.currentEngine.url, state.currentEngine.icon_path);
        nameEl.textContent = state.currentEngine.name;
    } else {
        // Show loading text in current language
        nameEl.textContent = i18n.t('app.loading');
    }
}

function renderEngineDropdown() {
    const dropdown = document.getElementById('engineDropdown');
    if (!dropdown) return;

    dropdown.innerHTML = state.engines.map(e => `
        <div class="engine-dropdown-item ${state.currentEngine?.id === e.id ? 'active' : ''}"
             data-engine-id="${e.id}">
            <img src="${getFavicon(e.url, e.icon_path)}" alt="" onerror="this.src='data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>🔍</text></svg>'">
            <span>${escapeHtml(e.name)}</span>
        </div>
    `).join('');

    // Bind click events
    dropdown.querySelectorAll('.engine-dropdown-item').forEach(item => {
        item.addEventListener('click', () => {
            const engineId = parseInt(item.dataset.engineId);
            state.currentEngine = state.engines.find(e => e.id === engineId);
            renderCurrentEngine();
            renderEngineDropdown();
            dropdown.classList.remove('show');
        });
    });
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
        return 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text y=".9em" font-size="90">🔍</text></svg>';
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
    renderGroups();
    updateGroupSelect();
    renderSettingsGroups();
}

function renderGroups() {
    document.querySelector('.groups-tabs').style.display = 'none';
}

function updateGroupSelect() {
    const select = document.getElementById('bookmarkGroup');
    select.innerHTML = `<option value="">${i18n.t('group.ungrouped')}</option>` +
        state.groups.map(g => `<option value="${g.id}">${escapeHtml(g.name)}</option>`).join('');
}

// ===================================
// Bookmarks
// ===================================
async function loadBookmarks() {
    state.bookmarks = await loadData(`${API}/bookmarks`, i18n.t('errors.loadBookmarkFailed'));
    renderBookmarks();
}

function renderBookmarks() {
    const container = document.getElementById('bookmarksContainer');

    if (state.bookmarks.length === 0) {
        container.innerHTML = `<div class="empty-state">${i18n.t('bookmark.empty')}</div>`;
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
    let html = '';
    sortedGroups.forEach(group => {
        const bookmarks = groupedBookmarks[group.id];
        if (bookmarks && bookmarks.length > 0) {
            html += `
                <div class="bookmark-group">
                    <h3 class="group-title">${escapeHtml(group.name)}</h3>
                    <div class="group-bookmarks">
                        ${bookmarks.map(renderBookmarkCard).join('')}
                    </div>
                </div>
            `;
        }
    });

    if (ungroupedBookmarks.length > 0) {
        html += `
            <div class="bookmark-group">
                <h3 class="group-title">${i18n.t('group.ungrouped')}</h3>
                <div class="group-bookmarks">
                    ${ungroupedBookmarks.map(renderBookmarkCard).join('')}
                </div>
            </div>
        `;
    }

    container.innerHTML = html;
    applyBookmarkSize();
    initDragAndDrop();
    initBookmarkClickHandlers();
}

// 初始化书签卡片点击事件
function initBookmarkClickHandlers() {
    const cards = document.querySelectorAll('.bookmark-card');
    cards.forEach(card => {
        card.addEventListener('click', (e) => {
            // 如果点击的是按钮，不处理
            if (e.target.closest('.bookmark-actions')) {
                return;
            }
            const url = card.dataset.url;
            const target = card.dataset.target || '_blank';
            if (url) {
                window.open(url, target);
            }
        });
    });
}

function renderBookmarkCard(b) {
    const target = b.is_new_window !== false ? '_blank' : '_self';
    return `
        <div class="bookmark-card" draggable="true" data-id="${b.id}" data-url="${escapeHtml(b.url)}" data-target="${target}">
            <img src="${getFavicon(b.url, b.icon_path || b.icon_url)}"
                 class="bookmark-icon"
                 onerror="this.src='data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>🌐</text></svg>'">
            <div class="bookmark-content">
                <div class="bookmark-title">${escapeHtml(b.title)}</div>
                ${b.description ? `<div class="bookmark-desc">${escapeHtml(b.description)}</div>` : ''}
                <div class="bookmark-url">${escapeHtml(b.url)}</div>
            </div>
            <div class="bookmark-actions">
                <button class="action-icon-btn" onclick="event.stopPropagation(); editBookmark(${b.id})" data-i18n-title="actions.edit" title="Edit">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                </button>
                <button class="action-icon-btn delete" onclick="event.stopPropagation(); deleteBookmark(${b.id})" data-i18n-title="actions.delete" title="Delete">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/>
                        <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                    </svg>
                </button>
            </div>
        </div>
    `;
}

// ===================================
// Drag and Drop
// ===================================
function initDraggable(selector, onDrop) {
    const items = document.querySelectorAll(selector);

    items.forEach(item => {
        item.addEventListener('dragstart', (e) => {
            state.draggedItem = item;
            item.classList.add('dragging');
        });

        item.addEventListener('dragend', () => {
            item.classList.remove('dragging');
            state.draggedItem = null;
        });

        item.addEventListener('dragover', (e) => {
            e.preventDefault();
            const dragging = document.querySelector('.dragging');
            if (dragging && dragging !== item) {
                const rect = item.getBoundingClientRect();
                const midpoint = rect.y + rect.height / 2;
                item.parentNode.insertBefore(dragging, e.clientY < midpoint ? item : item.nextSibling);
            }
        });

        item.addEventListener('drop', async (e) => {
            e.preventDefault();
            await onDrop();
        });
    });
}

function initDragAndDrop() {
    initDraggable('.bookmark-card', saveBookmarkOrder);
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
// Bookmark Management
// ===================================
function showBookmarkModal(bookmark = null) {
    const modal = document.getElementById('bookmarkModal');
    const title = document.getElementById('bookmarkModalTitle');

    if (bookmark) {
        title.textContent = i18n.t('bookmark.edit');
        document.getElementById('bookmarkId').value = bookmark.id;
        document.getElementById('bookmarkUrl').value = bookmark.url;
        document.getElementById('bookmarkTitle').value = bookmark.title;
        document.getElementById('bookmarkIcon').value = bookmark.icon_path || bookmark.icon_url || '';
        document.getElementById('bookmarkGroup').value = bookmark.group_id !== null && bookmark.group_id !== undefined ? bookmark.group_id : '';
        document.getElementById('bookmarkDesc').value = bookmark.description || '';
        document.getElementById('bookmarkNewWindow').checked = bookmark.is_new_window !== false; // 默认true

        // Show icon preview - use getFavicon to get correct icon
        const faviconUrl = getFavicon(bookmark.url, bookmark.icon_path || bookmark.icon_url);
        showIconPreview(faviconUrl);
    } else {
        title.textContent = i18n.t('bookmark.add');
        // 先设置默认值，再重置表单
        document.getElementById('bookmarkNewWindow').checked = true;
        document.getElementById('bookmarkForm').reset();
        document.getElementById('bookmarkId').value = '';
        document.getElementById('uploadFileName').textContent = '';
        hideIconPreview();
        // reset() 会重置 checkbox，所以需要再次设置
        document.getElementById('bookmarkNewWindow').checked = true;
    }

    modal.classList.add('show');
}

function showIconPreview(iconUrl) {
    const container = document.getElementById('iconPreviewContainer');
    const preview = document.getElementById('iconPreview');
    preview.src = iconUrl;
    preview.onerror = () => {
        preview.src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text y=".9em" font-size="90">🌐</text></svg>';
    };
    container.style.display = 'flex';
}

function hideIconPreview() {
    document.getElementById('iconPreviewContainer').style.display = 'none';
}

async function fetchIcon() {
    const url = document.getElementById('bookmarkUrl').value.trim();
    if (!url) {
        alert(i18n.t('bookmark.enterUrlFirst'));
        return;
    }

    const favicon = getFavicon(url);
    document.getElementById('bookmarkIcon').value = favicon;
    showIconPreview(favicon);
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
        await apiRequest(`${API}/group/${id}`, { method: 'DELETE' });
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
                <button class="btn btn-secondary btn-sm" onclick="editGroup(${g.id})">${i18n.t('actions.edit')}</button>
                <button class="btn btn-danger btn-sm" onclick="deleteGroup(${g.id})">${i18n.t('actions.delete')}</button>
            </div>
        </div>
    `).join('');

    // 绑定拖拽事件
    initGroupDragAndDrop();
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
                <button class="btn btn-secondary btn-sm" onclick="editEngine(${e.id})">${i18n.t('actions.edit')}</button>
                ${!e.is_default ? `<button class="btn btn-danger btn-sm" onclick="deleteEngine(${e.id})">${i18n.t('actions.delete')}</button>` : ''}
            </div>
        </div>
    `).join('');
}

// ===================================
// Import/Export
// ===================================
function showImportModal() {
    const modal = document.getElementById('importModal');
    document.getElementById('importFile').value = '';
    document.getElementById('importResult').className = 'import-result';
    document.getElementById('importResult').style.display = 'none';
    modal.classList.add('show');
}

async function importBookmarks() {
    const fileInput = document.getElementById('importFile');
    const result = document.getElementById('importResult');

    if (!fileInput.files.length) {
        alert(i18n.t('errors.noFileSelected'));
        return;
    }

    const formData = new FormData();
    formData.append('file', fileInput.files[0]);

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
            await loadBookmarks();
            await loadGroups();
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

    // Settings button
    document.getElementById('settingsBtn').addEventListener('click', async () => {
        await checkAuthStatus();
        renderSettingsEngines();
        i18n.updatePage();
        document.getElementById('settingsModal').classList.add('show');
    });

    // Background color selection
    document.querySelectorAll('input[name="bgColor"]').forEach(radio => {
        radio.addEventListener('change', (e) => {
            saveSetting('bgColor', e.target.value);
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
    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.classList.remove('show');
            }
        });
    });

    // Group management button
    document.getElementById('addGroupBtn').addEventListener('click', () => showGroupModal());

    // Add bookmark button
    document.getElementById('addBookmarkBtn').addEventListener('click', () => showBookmarkModal());

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
        document.getElementById('iconUpload').click();
    });

    // Icon upload
    document.getElementById('iconUpload').addEventListener('change', (e) => {
        const file = e.target.files[0];
        if (file) {
            uploadIcon(file);
        }
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

        const id = document.getElementById('bookmarkId').value;
        const groupValue = document.getElementById('bookmarkGroup').value;
        const data = {
            url: document.getElementById('bookmarkUrl').value,
            title: document.getElementById('bookmarkTitle').value,
            icon_url: document.getElementById('bookmarkIcon').value,
            group_id: groupValue === '' || groupValue === null || groupValue === undefined ? null : parseInt(groupValue, 10),
            description: document.getElementById('bookmarkDesc').value,
            is_new_window: document.getElementById('bookmarkNewWindow').checked
        };

        try {
            if (id) {
                await apiRequest(`${API}/bookmark/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify(data)
                });
            } else {
                await apiRequest(`${API}/bookmarks`, {
                    method: 'POST',
                    body: JSON.stringify(data)
                });
            }

            document.getElementById('bookmarkModal').classList.remove('show');
            await loadBookmarks();
            await loadGroups();
        } catch (err) {
            console.error(i18n.t('errors.saveFailed'), err);
            alert(i18n.t('errors.saveFailed'));
        }
    });

    // Group form submit
    document.getElementById('groupForm').addEventListener('submit', async (e) => {
        e.preventDefault();

        const id = document.getElementById('groupId').value;
        const data = {
            name: document.getElementById('groupName').value
        };

        try {
            if (id) {
                await apiRequest(`${API}/group/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify(data)
                });
            } else {
                await apiRequest(`${API}/groups`, {
                    method: 'POST',
                    body: JSON.stringify(data)
                });
            }

            document.getElementById('groupModal').classList.remove('show');
            await loadGroups();
        } catch (err) {
            console.error(i18n.t('errors.saveFailed'), err);
            alert(i18n.t('errors.saveFailed'));
        }
    });

    // Search engine form submit
    document.getElementById('engineEditForm').addEventListener('submit', async (e) => {
        e.preventDefault();

        const id = document.getElementById('engineId').value;
        const data = {
            name: document.getElementById('engineName').value,
            url: document.getElementById('engineUrl').value,
            placeholder: document.getElementById('enginePlaceholder').value,
            is_default: document.getElementById('engineDefault').checked
        };

        try {
            if (id) {
                await apiRequest(`${API}/search-engine/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify(data)
                });
            } else {
                await apiRequest(`${API}/search-engines`, {
                    method: 'POST',
                    body: JSON.stringify(data)
                });
            }

            document.getElementById('engineEditModal').classList.remove('show');
            await loadSearchEngines();
            renderSettingsEngines();
        } catch (err) {
            console.error(i18n.t('errors.saveFailed'), err);
            alert(i18n.t('errors.saveFailed'));
        }
    });

    // Window resize - 重新应用书签尺寸设置以适配桌面/移动端切换
    let resizeTimer;
    window.addEventListener('resize', () => {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(() => {
            applyBookmarkSize();
        }, 100);
    });
}

// HTML escape
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
