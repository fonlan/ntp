// API base path
const API = '/api';

// Helper function to make API requests with language header
async function apiRequest(url, options = {}) {
    const headers = {
        ...i18n.getRequestHeaders(),
        ...(options.headers || {})
    };

    // If body is provided and Content-Type is not set, default to JSON
    // But skip for FormData (let browser set the boundary)
    if (options.body && !(options.body instanceof FormData) && !headers['Content-Type']) {
        headers['Content-Type'] = 'application/json';
    }

    return fetch(url, {
        ...options,
        headers
    });
}

// Global state
let state = {
    bookmarks: [],
    groups: [],
    engines: [],
    currentGroup: null,
    currentEngine: null,
    draggedItem: null,
    bookmarkWidth: 320,
    bookmarkHeight: 80,
    bookmarkColumns: 0
};

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    // Wait for i18n to be ready
    await i18n.init();
    loadSettings();
    loadSearchEngines();
    loadGroups();
    loadBookmarks();
    initEventListeners();
});

// ===================================
// Settings
// ===================================
function loadSettings() {
    // Load background color
    const bgColor = localStorage.getItem('bgColor') || '#F8FAFC';
    document.body.style.background = bgColor;
    const bgColorRadio = document.querySelector(`input[name="bgColor"][value="${bgColor}"]`);
    if (bgColorRadio) bgColorRadio.checked = true;

    // Load bookmark size
    state.bookmarkWidth = parseInt(localStorage.getItem('bookmarkWidth')) || 320;
    state.bookmarkHeight = parseInt(localStorage.getItem('bookmarkHeight')) || 80;
    state.bookmarkColumns = parseInt(localStorage.getItem('bookmarkColumns')) || 0;

    // Update control values
    document.getElementById('bookmarkWidth').value = state.bookmarkWidth;
    document.getElementById('bookmarkHeight').value = state.bookmarkHeight;
    document.getElementById('bookmarkColumns').value = state.bookmarkColumns;

    // Update display values
    updateSizeDisplay();
    updatePreview();

    // Apply bookmark size style
    applyBookmarkSize();
}

function saveSetting(key, value) {
    localStorage.setItem(key, value);
    if (key === 'bgColor') {
        document.body.style.background = value;
    } else if (key === 'bookmarkWidth') {
        state.bookmarkWidth = parseInt(value);
        updateSizeDisplay();
        updatePreview();
        applyBookmarkSize();
    } else if (key === 'bookmarkHeight') {
        state.bookmarkHeight = parseInt(value);
        updateSizeDisplay();
        updatePreview();
        applyBookmarkSize();
    } else if (key === 'bookmarkColumns') {
        state.bookmarkColumns = parseInt(value);
        updateSizeDisplay();
        applyBookmarkSize();
    }
}

function updateSizeDisplay() {
    document.getElementById('widthValue').textContent = state.bookmarkWidth;
    document.getElementById('heightValue').textContent = state.bookmarkHeight;

    const columnsSelect = document.getElementById('bookmarkColumns');
    const columnsText = columnsSelect.options[columnsSelect.selectedIndex].text;
    document.getElementById('columnsValue').textContent = columnsText;
}

function updatePreview() {
    const preview = document.getElementById('bookmarkPreview');
    // Set outer dimensions of the card (including padding)
    preview.style.width = state.bookmarkWidth + 'px';
    preview.style.height = state.bookmarkHeight + 'px';
    preview.style.minHeight = state.bookmarkHeight + 'px';
}

function applyBookmarkSize() {
    const groupBookmarksContainers = document.querySelectorAll('.group-bookmarks');
    const cards = document.querySelectorAll('.bookmark-card');

    // 对每个分组容器设置列数
    groupBookmarksContainers.forEach(container => {
        if (state.bookmarkColumns > 0) {
            container.style.gridTemplateColumns = `repeat(${state.bookmarkColumns}, 1fr)`;
        } else {
            container.style.gridTemplateColumns = `repeat(auto-fill, minmax(${state.bookmarkWidth}px, 1fr))`;
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

// ===================================
// Search Engine
// ===================================
async function loadSearchEngines() {
    try {
        const res = await apiRequest(`${API}/search-engines`);
        state.engines = await res.json();
        state.currentEngine = state.engines.find(e => e.is_default) || state.engines[0];
        renderCurrentEngine();
        renderEngineDropdown();
    } catch (err) {
        console.error(i18n.t('errors.loadEngineFailed'), err);
    }
}

function renderCurrentEngine() {
    const iconEl = document.getElementById('currentEngineIcon');
    const nameEl = document.getElementById('currentEngineName');

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
    dropdown.classList.toggle('show');
}

// ===================================
// Groups
// ===================================
async function loadGroups() {
    try {
        const res = await apiRequest(`${API}/groups`);
        state.groups = await res.json();
        renderGroups();
        updateGroupSelect();
        renderSettingsGroups();
    } catch (err) {
        console.error(i18n.t('errors.loadGroupFailed'), err);
    }
}

function renderGroups() {
    // 移除分组标签栏，因为现在所有分组都会显示在首页
    const container = document.querySelector('.groups-tabs');
    container.style.display = 'none';
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
    try {
        // 加载所有书签，不再按分组过滤
        const res = await apiRequest(`${API}/bookmarks`);
        state.bookmarks = await res.json();
        renderBookmarks();
    } catch (err) {
        console.error(i18n.t('errors.loadBookmarkFailed'), err);
    }
}

function renderBookmarks() {
    const container = document.getElementById('bookmarksContainer');

    if (state.bookmarks.length === 0) {
        container.innerHTML = `<div class="empty-state">${i18n.t('bookmark.empty')}</div>`;
        return;
    }

    // 按分组分组书签，保持分组的排序顺序
    const groupedBookmarks = {};
    const ungroupedBookmarks = [];

    // 初始化所有分组（按 sort_order 排序）
    const sortedGroups = [...state.groups].sort((a, b) => a.sort_order - b.sort_order);
    sortedGroups.forEach(g => {
        groupedBookmarks[g.id] = [];
    });

    // 分配书签到对应分组
    state.bookmarks.forEach(b => {
        if (b.group_id) {
            if (groupedBookmarks[b.group_id]) {
                groupedBookmarks[b.group_id].push(b);
            }
        } else {
            ungroupedBookmarks.push(b);
        }
    });

    // 渲染 HTML
    let html = '';

    // 渲染每个分组
    sortedGroups.forEach(group => {
        const bookmarks = groupedBookmarks[group.id];
        if (bookmarks && bookmarks.length > 0) {
            html += `
                <div class="bookmark-group">
                    <h3 class="group-title">${escapeHtml(group.name)}</h3>
                    <div class="group-bookmarks">
                        ${bookmarks.map(b => renderBookmarkCard(b)).join('')}
                    </div>
                </div>
            `;
        }
    });

    // 渲染未分组书签
    if (ungroupedBookmarks.length > 0) {
        html += `
            <div class="bookmark-group">
                <h3 class="group-title">${i18n.t('group.ungrouped')}</h3>
                <div class="group-bookmarks">
                    ${ungroupedBookmarks.map(b => renderBookmarkCard(b)).join('')}
                </div>
            </div>
        `;
    }

    container.innerHTML = html;

    // Apply bookmark size and bind drag events
    applyBookmarkSize();
    initDragAndDrop();
}

function renderBookmarkCard(b) {
    return `
        <div class="bookmark-card" draggable="true" data-id="${b.id}">
            <img src="${getFavicon(b.url, b.icon_path || b.icon_url)}"
                 class="bookmark-icon"
                 onerror="this.src='data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>🌐</text></svg>'">
            <div class="bookmark-content">
                <div class="bookmark-title">${escapeHtml(b.title)}</div>
                ${b.description ? `<div class="bookmark-desc">${escapeHtml(b.description)}</div>` : ''}
                <div class="bookmark-url">${escapeHtml(b.url)}</div>
            </div>
            <div class="bookmark-actions">
                <button class="action-icon-btn" onclick="editBookmark(${b.id})" data-i18n-title="actions.edit" title="Edit">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                </button>
                <button class="action-icon-btn delete" onclick="deleteBookmark(${b.id})" data-i18n-title="actions.delete" title="Delete">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/>
                        <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                    </svg>
                </button>
            </div>
        </div>
    `;
}

// Drag and drop sorting
function initDragAndDrop() {
    const cards = document.querySelectorAll('.bookmark-card');

    cards.forEach(card => {
        card.addEventListener('dragstart', (e) => {
            state.draggedItem = card;
            card.classList.add('dragging');
        });

        card.addEventListener('dragend', () => {
            card.classList.remove('dragging');
            state.draggedItem = null;
        });

        card.addEventListener('dragover', (e) => {
            e.preventDefault();
            const dragging = document.querySelector('.dragging');
            if (dragging && dragging !== card) {
                const rect = card.getBoundingClientRect();
                const midpoint = rect.y + rect.height / 2;
                if (e.clientY < midpoint) {
                    card.parentNode.insertBefore(dragging, card);
                } else {
                    card.parentNode.insertBefore(dragging, card.nextSibling);
                }
            }
        });

        card.addEventListener('drop', async (e) => {
            e.preventDefault();
            await saveBookmarkOrder();
        });
    });
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

        // Show icon preview
        if (bookmark.icon_path || bookmark.icon_url) {
            showIconPreview(bookmark.icon_path || bookmark.icon_url);
        }
    } else {
        title.textContent = i18n.t('bookmark.add');
        document.getElementById('bookmarkForm').reset();
        document.getElementById('bookmarkId').value = '';
        document.getElementById('uploadFileName').textContent = '';
        hideIconPreview();
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
    if (bookmark) {
        showBookmarkModal(bookmark);
    }
}

async function deleteBookmark(id) {
    if (!confirm(i18n.t('bookmark.deleteConfirm'))) return;

    try {
        await apiRequest(`${API}/bookmark/${id}`, { method: 'DELETE' });
        await loadBookmarks();
        await loadGroups();
    } catch (err) {
        console.error(i18n.t('errors.deleteFailed'), err);
        alert(i18n.t('errors.deleteFailed'));
    }
}

// ===================================
// Group Management
// ===================================
function showGroupModal(group = null) {
    const modal = document.getElementById('groupModal');
    const title = document.getElementById('groupModalTitle');

    if (group) {
        title.textContent = i18n.t('group.edit');
        document.getElementById('groupId').value = group.id;
        document.getElementById('groupName').value = group.name;
    } else {
        title.textContent = i18n.t('group.add');
        document.getElementById('groupForm').reset();
        document.getElementById('groupId').value = '';
    }

    modal.classList.add('show');
}

async function editGroup(id) {
    const group = state.groups.find(g => g.id === id);
    if (group) {
        showGroupModal(group);
    }
}

async function deleteGroup(id) {
    if (!confirm(i18n.t('group.deleteConfirm'))) return;

    try {
        await apiRequest(`${API}/group/${id}`, { method: 'DELETE' });
        await loadGroups();
        renderSettingsGroups();
    } catch (err) {
        console.error(i18n.t('errors.deleteFailed'), err);
        alert(i18n.t('errors.deleteFailed'));
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
    const items = document.querySelectorAll('#groupsList .settings-item');

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
            const dragging = document.querySelector('#groupsList .dragging');
            if (dragging && dragging !== item) {
                const rect = item.getBoundingClientRect();
                const midpoint = rect.y + rect.height / 2;
                if (e.clientY < midpoint) {
                    item.parentNode.insertBefore(dragging, item);
                } else {
                    item.parentNode.insertBefore(dragging, item.nextSibling);
                }
            }
        });

        item.addEventListener('drop', async (e) => {
            e.preventDefault();
            await saveGroupOrder();
        });
    });
}

async function saveGroupOrder() {
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
        // 重新加载分组以获取更新后的排序
        await loadGroups();
    } catch (err) {
        console.error(i18n.t('errors.saveOrderFailed'), err);
        alert(i18n.t('errors.saveOrderFailed'));
    }
}

// ===================================
// Search Engine Management
// ===================================
function showEngineModal(engine = null) {
    const modal = document.getElementById('engineEditModal');
    const title = document.getElementById('engineEditModalTitle');

    if (engine) {
        title.textContent = i18n.t('searchEngine.edit');
        document.getElementById('engineId').value = engine.id;
        document.getElementById('engineName').value = engine.name;
        document.getElementById('engineUrl').value = engine.url;
        document.getElementById('enginePlaceholder').value = engine.placeholder || '';
        document.getElementById('engineDefault').checked = engine.is_default;
    } else {
        title.textContent = i18n.t('searchEngine.add');
        document.getElementById('engineEditForm').reset();
        document.getElementById('engineId').value = '';
    }

    modal.classList.add('show');
}

async function editEngine(id) {
    const engine = state.engines.find(e => e.id === id);
    if (engine) {
        showEngineModal(engine);
    }
}

async function deleteEngine(id) {
    if (!confirm(i18n.t('searchEngine.deleteConfirm'))) return;

    try {
        await apiRequest(`${API}/search-engine/${id}`, { method: 'DELETE' });
        await loadSearchEngines();
        renderSettingsEngines();
    } catch (err) {
        console.error(i18n.t('errors.deleteFailed'), err);
        alert(i18n.t('errors.deleteFailed'));
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
    document.getElementById('settingsBtn').addEventListener('click', () => {
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
    document.getElementById('bookmarkWidth').addEventListener('input', (e) => {
        saveSetting('bookmarkWidth', e.target.value);
    });

    document.getElementById('bookmarkHeight').addEventListener('input', (e) => {
        saveSetting('bookmarkHeight', e.target.value);
    });

    document.getElementById('bookmarkColumns').addEventListener('change', (e) => {
        saveSetting('bookmarkColumns', e.target.value);
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
            // Re-render components to apply new language
            renderGroups();
            renderBookmarks();
            updateGroupSelect();
            renderCurrentEngine();
            renderEngineDropdown();
            // Update modal titles if they're visible
            updateAllModalTitles();
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
            description: document.getElementById('bookmarkDesc').value
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
}

// HTML escape
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Update all modal titles when language changes
function updateAllModalTitles() {
    // This function is called when language is changed
    // Modals will update their titles when opened next time
}
