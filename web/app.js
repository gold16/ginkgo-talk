// ============================================
// Ginkgo Talk - AI 手机键盘
// ============================================

(function () {
    'use strict';

    // ---- DOM ----
    const micBtn = document.getElementById('micBtn');
    const micIcon = document.getElementById('micIcon');
    const stopIcon = document.getElementById('stopIcon');
    const inputText = document.getElementById('inputText');
    const charCount = document.getElementById('charCount');
    const sendBtn = document.getElementById('sendBtn');
    const clearPcBtn = document.getElementById('clearPcBtn');
    const aiStatus = document.getElementById('aiStatus');
    const enterBtn = document.getElementById('enterBtn');
    const shiftEnterBtn = document.getElementById('shiftEnterBtn');
    const ctrlZBtn = document.getElementById('ctrlZBtn');
    const tabBtn = document.getElementById('tabBtn');
    const ctrlVBtn = document.getElementById('ctrlVBtn');
    const escBtn = document.getElementById('escBtn');
    const statusBar = document.getElementById('statusBar');
    const statusText = document.getElementById('statusText');
    const pairCard = document.getElementById('pairCard');
    const pairMessage = document.getElementById('pairMessage');
    const pairCodeInput = document.getElementById('pairCodeInput');
    const pairSubmitBtn = document.getElementById('pairSubmitBtn');
    const pairHint = document.getElementById('pairHint');
    const historyList = document.getElementById('historyList');
    const clearBtn = document.getElementById('clearBtn');
    const modeBtns = document.querySelectorAll('.mode-btn');
    const langSelect = document.getElementById('langSelect');

    // ---- State ----
    let ws = null;
    let recognition = null;
    let isListening = false;
    let aiAvailable = false;
    let history = [];
    let aiProcessing = false;
    let reconnectTimer = null;
    let wsConnectTimeout = null;
    let isPaired = false;
    let pairSubmitting = false;
    let authToken = '';
    const deviceId = getOrCreateDeviceId();
    let sendTimeout = null;
    let currentLang = 'zh-CN';

    const I18N = {
        'zh-CN': {
            app: { title: 'Ginkgo Talk - AI 手机键盘', tagline: 'AI 手机键盘', install: '添加到主屏幕' },
            status: {
                connecting: '连接中...',
                connected: '已连接',
                disconnected: '已断开',
                connectFailed: '连接失败',
                connectTimeout: '连接超时，请检查网络',
                connectError: '连接错误',
                authExpired: '授权已失效，请输入配对码',
                pairNeedCode: '请输入配对码完成连接',
                pairUnavailable: '配对服务不可用',
                pairTimeout: '配对服务连接超时',
                pairRequestTimeout: '配对请求超时',
                micDenied: '麦克风权限被拒绝',
            },
            pair: {
                title: '设备配对',
                inputPlaceholder: '请输入 4 位配对码',
                confirm: '确认配对',
                submitting: '配对中...',
                hintNeedCode: '请输入电脑终端显示的 4 位配对码',
                hintNeedScan: '请使用电脑端二维码重新扫码打开页面',
                msgNeedCodeConnect: '请输入 4 位配对码完成连接。',
                msgAuthExpiredNeedCode: '链接授权已失效，请输入 4 位配对码。',
                msgServiceUnavailable: '配对服务不可用，请重试。',
                msgNeedCode: '请输入 4 位配对码。',
                msgServiceConnectFailed: '配对服务连接失败，请检查网络。',
                msgCodeInvalidFormat: '配对码必须是 4 位数字。',
                msgCodeInvalid: '配对码错误，请重试。',
                msgRequestFailed: '配对请求失败，请重试。',
            },
            input: { placeholder: '在这里输入文字，使用手机键盘或语音...', voiceTitle: 'Web Speech API 语音输入' },
            send: { send: '发送', sending: '发送中...' },
            shortcut: {
                title: '电脑端快捷键',
                enterTitle: '发送/确认 (Enter)',
                shiftEnterTitle: '换行 (Shift+Enter)',
                newLine: '换行',
                clearTitle: '清空电脑输入框 (Ctrl+A, Delete)',
                clear: '清除',
                undoTitle: '撤销 (Ctrl+Z)',
                undo: '撤销',
                tabTitle: 'Tab 切换焦点',
                pasteTitle: '粘贴 (Ctrl+V)',
                paste: '粘贴',
                escTitle: 'Escape 取消',
            },
            mode: {
                title: 'AI 工具',
                tidyTitle: '去重、去口头禅、加标点',
                tidy: '整理',
                formalTitle: '口语转书面语',
                formal: '正式',
                translateTitle: '中英自动互译',
                translate: '翻译',
            },
            ai: {
                disabledHint: 'AI 未启用，请先配置 API Key',
                done: '已{mode}，可编辑后发送',
                failed: 'AI 处理失败',
                processing: 'AI 处理中...',
                timeout: '处理超时，请重试',
            },
            history: {
                title: '发送记录',
                clear: '清空',
                empty: '还没有发送记录',
                original: '原文',
                sent: '已输入',
                sending: '发送中...',
                processing: 'AI 处理中...',
                preview: '已处理，待发送',
                aiError: 'AI 错误',
                error: '错误',
                modeRaw: '原始',
                modeTidy: '整理',
                modeFormal: '正式',
                modeTranslate: '翻译',
            },
            settings: {
                title: '⚙️ AI 设置',
                lanIpLabel: 'LAN IP（或 auto）',
                save: '保存',
                saving: '保存中...',
                needOneField: '请至少填写一项',
                saveOk: '已保存',
                saveOkAiOn: '已保存，AI 已启用',
                saveFailed: '保存失败',
                networkError: '网络错误',
            },
            pwa: { installHint: '请在浏览器菜单中选择“添加到主屏幕”或“安装应用”' },
        },
        'en-US': {
            app: { title: 'Ginkgo Talk - AI Mobile Keyboard', tagline: 'AI Mobile Keyboard', install: 'Add to Home Screen' },
            status: {
                connecting: 'Connecting...',
                connected: 'Connected',
                disconnected: 'Disconnected',
                connectFailed: 'Connection failed',
                connectTimeout: 'Connection timeout, please check network',
                connectError: 'Connection error',
                authExpired: 'Authorization expired, please enter pair code',
                pairNeedCode: 'Enter pair code to continue',
                pairUnavailable: 'Pairing service unavailable',
                pairTimeout: 'Pairing service timeout',
                pairRequestTimeout: 'Pair request timeout',
                micDenied: 'Microphone permission denied',
            },
            pair: {
                title: 'Device Pairing',
                inputPlaceholder: 'Enter 4-digit pair code',
                confirm: 'Confirm Pairing',
                submitting: 'Pairing...',
                hintNeedCode: 'Enter the 4-digit code shown on desktop terminal',
                hintNeedScan: 'Use desktop QR code to rescan and open this page',
                msgNeedCodeConnect: 'Enter 4-digit pair code to continue.',
                msgAuthExpiredNeedCode: 'Link authorization expired, enter 4-digit pair code.',
                msgServiceUnavailable: 'Pairing service unavailable, please retry.',
                msgNeedCode: 'Please enter the 4-digit pair code.',
                msgServiceConnectFailed: 'Pairing service connection failed, check your network.',
                msgCodeInvalidFormat: 'Pair code must be 4 digits.',
                msgCodeInvalid: 'Invalid pair code, please retry.',
                msgRequestFailed: 'Pair request failed, please retry.',
            },
            input: { placeholder: 'Type here using your phone keyboard or voice...', voiceTitle: 'Web Speech API Voice Input' },
            send: { send: 'Send', sending: 'Sending...' },
            shortcut: {
                title: 'Desktop Shortcuts',
                enterTitle: 'Send/Confirm (Enter)',
                shiftEnterTitle: 'New line (Shift+Enter)',
                newLine: 'New line',
                clearTitle: 'Clear desktop input (Ctrl+A, Delete)',
                clear: 'Clear',
                undoTitle: 'Undo (Ctrl+Z)',
                undo: 'Undo',
                tabTitle: 'Tab switch focus',
                pasteTitle: 'Paste (Ctrl+V)',
                paste: 'Paste',
                escTitle: 'Escape cancel',
            },
            mode: {
                title: 'AI Tools',
                tidyTitle: 'Deduplicate, remove fillers, add punctuation',
                tidy: 'Tidy',
                formalTitle: 'Convert speech to formal writing',
                formal: 'Formal',
                translateTitle: 'Auto translate between Chinese and English',
                translate: 'Translate',
            },
            ai: {
                disabledHint: 'AI not enabled, configure API Key first',
                done: '{mode} done, edit then send',
                failed: 'AI processing failed',
                processing: 'AI processing...',
                timeout: 'Processing timeout, please retry',
            },
            history: {
                title: 'Send History',
                clear: 'Clear',
                empty: 'No send history yet',
                original: 'Original',
                sent: 'Typed',
                sending: 'Sending...',
                processing: 'AI processing...',
                preview: 'Processed, pending send',
                aiError: 'AI error',
                error: 'Error',
                modeRaw: 'Raw',
                modeTidy: 'Tidy',
                modeFormal: 'Formal',
                modeTranslate: 'Translate',
            },
            settings: {
                title: '⚙️ AI Settings',
                lanIpLabel: 'LAN IP (or auto)',
                save: 'Save',
                saving: 'Saving...',
                needOneField: 'Please fill at least one field',
                saveOk: 'Saved',
                saveOkAiOn: 'Saved, AI enabled',
                saveFailed: 'Save failed',
                networkError: 'Network error',
            },
            pwa: { installHint: 'Use browser menu to choose "Add to Home Screen" or "Install App"' },
        },
    };

    function t(key, vars) {
        const dict = I18N[currentLang] || I18N['zh-CN'];
        const fallback = I18N['zh-CN'];
        let val = key.split('.').reduce((acc, k) => (acc && acc[k] !== undefined ? acc[k] : undefined), dict);
        if (val === undefined) {
            val = key.split('.').reduce((acc, k) => (acc && acc[k] !== undefined ? acc[k] : undefined), fallback);
        }
        if (typeof val !== 'string') return key;
        if (!vars) return val;
        return val.replace(/\{(\w+)\}/g, (_, name) => (vars[name] !== undefined ? String(vars[name]) : `{${name}}`));
    }

    function applyI18n() {
        document.documentElement.lang = currentLang;
        document.querySelectorAll('[data-i18n]').forEach((el) => {
            el.textContent = t(el.getAttribute('data-i18n'));
        });
        document.querySelectorAll('[data-i18n-placeholder]').forEach((el) => {
            el.setAttribute('placeholder', t(el.getAttribute('data-i18n-placeholder')));
        });
        document.querySelectorAll('[data-i18n-title]').forEach((el) => {
            el.setAttribute('title', t(el.getAttribute('data-i18n-title')));
        });
        if (langSelect) langSelect.value = currentLang;
        if (pairSubmitBtn && !pairSubmitting) pairSubmitBtn.textContent = t('pair.confirm');
        if (sendBtn && !sendBtn.disabled) sendBtn.querySelector('span').textContent = t('send.send');
    }

    function initLanguage() {
        const saved = localStorage.getItem('gtalk_lang');
        if (saved && I18N[saved]) {
            currentLang = saved;
        } else {
            currentLang = (navigator.language || '').toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US';
        }
        applyI18n();
    }

    function setLanguage(lang) {
        if (!I18N[lang]) return;
        currentLang = lang;
        localStorage.setItem('gtalk_lang', lang);
        applyI18n();
        if (recognition) recognition.lang = currentLang;
        renderHistory();
        updateModeButtons();
    }

    function modeLabel(mode) {
        if (mode === 'tidy') return t('history.modeTidy');
        if (mode === 'formal') return t('history.modeFormal');
        if (mode === 'translate') return t('history.modeTranslate');
        return t('history.modeRaw');
    }

    function getOrCreateDeviceId() {
        const key = 'gtalk_device_id';
        let id = localStorage.getItem(key);
        if (id) return id;

        if (window.crypto && window.crypto.getRandomValues) {
            const bytes = new Uint8Array(16);
            window.crypto.getRandomValues(bytes);
            id = Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
        } else {
            id = `dev_${Date.now()}_${Math.random().toString(16).slice(2)}`;
        }
        localStorage.setItem(key, id);
        return id;
    }

    function initAuthToken() {
        const tokenInUrl = (new URLSearchParams(window.location.search).get('token') || '').trim();
        const tokenInStorage = (localStorage.getItem('gtalk_auth_token') || '').trim();
        if (tokenInUrl) {
            authToken = tokenInUrl;
            localStorage.setItem('gtalk_auth_token', tokenInUrl);
            return;
        }
        authToken = tokenInStorage;
    }

    function setAuthToken(token) {
        authToken = (token || '').trim();
        if (authToken) {
            localStorage.setItem('gtalk_auth_token', authToken);
        } else {
            localStorage.removeItem('gtalk_auth_token');
        }
    }

    function withAuth(url) {
        const params = new URLSearchParams();
        if (authToken) params.set('token', authToken);
        if (deviceId) params.set('device_id', deviceId);
        const qs = params.toString();
        if (!qs) return url;
        const sep = url.includes('?') ? '&' : '?';
        return `${url}${sep}${qs}`;
    }

    async function fetchWithTimeout(url, options, timeoutMs) {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), timeoutMs || 8000);
        try {
            return await fetch(url, { ...(options || {}), signal: controller.signal });
        } finally {
            clearTimeout(timeout);
        }
    }

    function showPairCard(message, needCode) {
        pairCard.classList.remove('hidden');
        pairMessage.textContent = message || '';
        pairCodeInput.classList.toggle('hidden', !needCode);
        pairSubmitBtn.classList.toggle('hidden', !needCode);
        pairHint.textContent = needCode
            ? t('pair.hintNeedCode')
            : t('pair.hintNeedScan');
        if (needCode) pairCodeInput.focus();
    }

    function hidePairCard() {
        pairCard.classList.add('hidden');
    }

    function setStatus(state, text) {
        statusBar.className = 'status-bar ' + state;
        statusText.textContent = text;
    }

    // ---- Pairing ----
        async function ensurePaired() {
        if (!authToken) {
            setStatus('error', t('status.pairNeedCode'));
            showPairCard(t('pair.msgNeedCodeConnect'), true);
            return false;
        }

        try {
            const resp = await fetchWithTimeout(withAuth('/api/pair'), null, 8000);
            if (resp.status === 401) {
                setAuthToken('');
                setStatus('error', t('status.authExpired'));
                showPairCard(t('pair.msgAuthExpiredNeedCode'), true);
                return false;
            }
            if (!resp.ok) {
                setStatus('error', t('status.pairUnavailable'));
                showPairCard(t('pair.msgServiceUnavailable'), true);
                return false;
            }
            const data = await resp.json();
            if (data.paired) {
                isPaired = true;
                hidePairCard();
                return true;
            }
            showPairCard(t('pair.msgNeedCode'), true);
            return false;
        } catch (e) {
            setStatus('error', t('status.pairTimeout'));
            showPairCard(t('pair.msgServiceConnectFailed'), true);
            return false;
        }
    }

        async function submitPairCode() {
        if (pairSubmitting) return;
        const code = (pairCodeInput.value || '').trim();
        if (!/^\d{4}$/.test(code)) {
            showPairCard(t('pair.msgCodeInvalidFormat'), true);
            return;
        }

        pairSubmitting = true;
        pairSubmitBtn.disabled = true;
        pairSubmitBtn.textContent = t('pair.submitting');
        try {
            const resp = await fetchWithTimeout(withAuth('/api/pair'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ code, deviceId }),
            }, 8000);
            if (!resp.ok) {
                showPairCard(t('pair.msgCodeInvalid'), true);
                return;
            }
            const data = await resp.json();
            if (data && data.token) {
                setAuthToken(data.token);
            }
            isPaired = true;
            hidePairCard();
            connectWebSocket();
        } catch (e) {
            setStatus('error', t('status.pairRequestTimeout'));
            showPairCard(t('pair.msgRequestFailed'), true);
        } finally {
            pairSubmitting = false;
            pairSubmitBtn.disabled = false;
            pairSubmitBtn.textContent = t('pair.confirm');
        }
    }

    // ---- WebSocket ----
    async function connectWebSocket() {
        setStatus('', t('status.connecting'));
        if (!(await ensurePaired())) return;

        if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
            return;
        }

        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${location.host}${withAuth('/ws')}`;

        try {
            ws = new WebSocket(wsUrl);
        } catch (e) {
            setStatus('error', t('status.connectFailed'));
            scheduleReconnect();
            return;
        }

        clearTimeout(wsConnectTimeout);
        wsConnectTimeout = setTimeout(() => {
            if (ws && ws.readyState === WebSocket.CONNECTING) {
                try { ws.close(); } catch (e) { }
                setStatus('error', t('status.connectTimeout'));
            }
        }, 8000);

        ws.onopen = () => {
            setStatus('connected', t('status.connected'));
            clearTimeout(wsConnectTimeout);
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
            hidePairCard();
            fetchStatus();
        };

        ws.onclose = () => {
            clearTimeout(wsConnectTimeout);
            setStatus('', t('status.disconnected'));
            ws = null;
            scheduleReconnect();
        };

        ws.onerror = () => {
            clearTimeout(wsConnectTimeout);
            setStatus('error', t('status.connectError'));
        };

        ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                switch (msg.type) {
                    case 'ack':
                        if (msg.original && msg.text !== msg.original && msg.mode !== 'raw') {
                            updateLastHistory(msg.text, msg.original, 'sent');
                        } else {
                            updateLastHistoryStatus('sent');
                        }
                        enableSend();
                        break;
                    case 'ai_preview': {
                        aiProcessing = false;
                        inputText.disabled = false;
                        inputText.value = msg.text;
                        updateCharCount();
                        updateLastHistory(msg.text, msg.original, 'preview');
                        clearTimeout(sendTimeout);
                        modeBtns.forEach(b => b.classList.remove('disabled'));
                        updateModeButtons();
                        const modeLabels = { tidy: t('mode.tidy'), formal: t('mode.formal'), translate: t('mode.translate') };
                        showAIStatus('done', t('ai.done', { mode: modeLabels[msg.mode] || 'OK' }));
                        inputText.focus();
                        break;
                    }
                    case 'processing':
                        updateLastHistoryStatus('processing');
                        break;
                    case 'ai_error':
                        aiProcessing = false;
                        inputText.disabled = false;
                        updateLastHistoryStatus('ai_error', msg.error);
                        enableSend();
                        modeBtns.forEach(b => b.classList.remove('disabled'));
                        updateModeButtons();
                        showAIStatus('error', t('ai.failed'));
                        break;
                    case 'error':
                        updateLastHistoryStatus('error', msg.error);
                        enableSend();
                        break;
                }
            } catch (e) {
                console.error('Bad message:', e);
            }
        };
    }

    function scheduleReconnect() {
        if (reconnectTimer) return;
        reconnectTimer = setTimeout(() => {
            reconnectTimer = null;
            connectWebSocket();
        }, 3000);
    }

    function fetchStatus() {
        fetch(withAuth('/api/status'))
            .then(r => r.json())
            .then(data => {
                isPaired = !!data.paired;
                aiAvailable = data.aiAvailable;
                updateModeButtons();
            })
            .catch(() => { });
    }

    function sendText(text) {
        if (ws && ws.readyState === WebSocket.OPEN && text.trim()) {
            ws.send(JSON.stringify({ type: 'text', text: text.trim(), mode: 'raw' }));
            return true;
        }
        return false;
    }

    function sendAIProcess(text, mode) {
        if (ws && ws.readyState === WebSocket.OPEN && text.trim()) {
            ws.send(JSON.stringify({ type: 'text', text: text.trim(), mode }));
            return true;
        }
        return false;
    }

    function sendCommand(cmd) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'command', text: cmd }));
        }
    }

    // ---- UI ----
    function enableSend() {
        sendBtn.disabled = false;
        sendBtn.querySelector('span').textContent = t('send.send');
        clearTimeout(sendTimeout);
    }

    function showAIStatus(type, text) {
        aiStatus.textContent = text;
        aiStatus.className = 'ai-status ' + type;
        aiStatus.classList.remove('hidden');
    }

    function hideAIStatus() {
        aiStatus.classList.add('hidden');
        aiStatus.textContent = '';
    }

    function updateCharCount() {
        const len = inputText.value.length;
        charCount.textContent = len > 0 ? len : '';
    }

    function updateModeButtons() {
        modeBtns.forEach(btn => {
            if (!aiAvailable) {
                btn.classList.add('disabled');
                btn.title = t('ai.disabledHint');
            } else {
                btn.classList.remove('disabled');
            }
        });
    }

    // ---- Send ----
    function doSend() {
        const text = inputText.value.trim();
        if (!text) return;

        const sent = sendText(text);
        addHistory(text, sent ? 'sending' : 'error', 'raw');

        sendBtn.disabled = true;
        sendBtn.querySelector('span').textContent = t('send.sending');
        inputText.value = '';
        updateCharCount();
        inputText.focus();
        hideAIStatus();

        clearTimeout(sendTimeout);
        sendTimeout = setTimeout(() => { enableSend(); }, 15000);
    }

    function doAIProcess(mode) {
        const text = inputText.value.trim();
        if (!text || !aiAvailable || aiProcessing) return;

        aiProcessing = true;
        sendAIProcess(text, mode);
        addHistory(text, 'processing', mode);

        modeBtns.forEach(b => b.classList.add('disabled'));
        inputText.value = '';
        inputText.disabled = true;
        showAIStatus('processing', t('ai.processing'));

        clearTimeout(sendTimeout);
        sendTimeout = setTimeout(() => {
            aiProcessing = false;
            inputText.disabled = false;
            inputText.value = text;
            modeBtns.forEach(b => b.classList.remove('disabled'));
            updateModeButtons();
            showAIStatus('error', t('ai.timeout'));
        }, 20000);
    }

    // ---- Speech ----
    function initRecognition() {
        const SR = window.SpeechRecognition || window.webkitSpeechRecognition;
        if (!SR) {
            micBtn.style.display = 'none';
            return;
        }

        recognition = new SR();
        recognition.continuous = true;
        recognition.interimResults = true;
        recognition.lang = currentLang;

        recognition.onstart = () => {
            isListening = true;
            micBtn.classList.add('active');
            micIcon.classList.add('hidden');
            stopIcon.classList.remove('hidden');
        };

        recognition.onresult = (event) => {
            for (let i = event.resultIndex; i < event.results.length; i++) {
                if (event.results[i].isFinal) {
                    inputText.value += event.results[i][0].transcript;
                    updateCharCount();
                    inputText.scrollTop = inputText.scrollHeight;
                }
            }
        };

        recognition.onerror = (event) => {
            if (event.error === 'not-allowed') {
                setStatus('error', t('status.micDenied'));
                stopListening();
            } else if (event.error !== 'no-speech' && event.error !== 'aborted' && isListening) {
                setTimeout(() => { try { recognition.start(); } catch (e) { } }, 500);
            }
        };

        recognition.onend = () => {
            if (isListening) {
                try { recognition.start(); } catch (e) { stopListening(); }
            }
        };
    }

    function startListening() {
        if (!recognition) initRecognition();
        if (!recognition) return;
        try { recognition.start(); } catch (e) {
            recognition.stop();
            setTimeout(() => { try { recognition.start(); } catch (e2) { } }, 300);
        }
    }

    function stopListening() {
        isListening = false;
        micBtn.classList.remove('active');
        micIcon.classList.remove('hidden');
        stopIcon.classList.add('hidden');
        if (recognition) try { recognition.stop(); } catch (e) { }
    }

    // ---- History ----
    function addHistory(text, status, mode) {
        history.unshift({
            text,
            processed: null,
            original: text,
            status,
            mode: mode || 'raw',
            time: new Date().toLocaleTimeString(currentLang, { hour: '2-digit', minute: '2-digit', second: '2-digit' })
        });
        if (history.length > 50) history.pop();
        renderHistory();
    }

    function updateLastHistoryStatus(status, error) {
        if (history.length > 0) {
            history[0].status = status;
            if (error) history[0].error = error;
            renderHistory();
        }
    }

    function updateLastHistory(processedText, originalText, status) {
        if (history.length > 0) {
            history[0].processed = processedText;
            history[0].original = originalText;
            history[0].status = status;
            renderHistory();
        }
    }

    function statusLabel(status) {
        if (status === 'sent') return t('history.sent');
        if (status === 'sending') return t('history.sending');
        if (status === 'processing') return t('history.processing');
        if (status === 'preview') return t('history.preview');
        if (status === 'ai_error') return t('history.aiError');
        if (status === 'error') return t('history.error');
        return '...';
    }

    function renderHistory() {
        if (!history.length) {
            historyList.innerHTML = `<div class="history-empty">${esc(t('history.empty'))}</div>`;
            return;
        }

        historyList.innerHTML = history.map(item => {
            const hasProcessed = item.processed && item.processed !== item.original;
            return `
            <div class="history-item">
                ${hasProcessed
                    ? `<div class="history-text">${esc(item.processed)}</div><div class="history-original">${esc(t('history.original'))}: ${esc(item.original)}</div>`
                    : `<div class="history-text">${esc(item.text)}</div>`}
                <div class="history-meta">
                    <span>${item.time}</span>
                    ${item.mode !== 'raw' ? `<span class="history-mode">${modeLabel(item.mode)}</span>` : ''}
                    <span class="history-status ${item.status}">${statusLabel(item.status)}</span>
                </div>
            </div>`;
        }).join('');
    }

    function esc(t) {
        const d = document.createElement('div');
        d.textContent = t;
        return d.innerHTML;
    }

    // ---- Events ----
    sendBtn.addEventListener('click', doSend);
    inputText.addEventListener('input', updateCharCount);
    inputText.addEventListener('keydown', e => {
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
            e.preventDefault();
            doSend();
        }
    });

    pairSubmitBtn.addEventListener('click', submitPairCode);
    pairCodeInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            submitPairCode();
        }
    });

    micBtn.addEventListener('click', () => isListening ? stopListening() : startListening());
    clearPcBtn.addEventListener('click', () => sendCommand('clear'));
    enterBtn.addEventListener('click', () => sendCommand('enter'));
    shiftEnterBtn.addEventListener('click', () => sendCommand('shift_enter'));
    ctrlZBtn.addEventListener('click', () => sendCommand('ctrl_z'));
    tabBtn.addEventListener('click', () => sendCommand('tab'));
    ctrlVBtn.addEventListener('click', () => sendCommand('ctrl_v'));
    escBtn.addEventListener('click', () => sendCommand('escape'));
    clearBtn.addEventListener('click', () => {
        history = [];
        renderHistory();
    });

    modeBtns.forEach(btn => btn.addEventListener('click', () => {
        if (btn.classList.contains('disabled')) return;
        doAIProcess(btn.dataset.mode);
    }));

    if (langSelect) {
        langSelect.addEventListener('change', () => {
            setLanguage(langSelect.value);
        });
    }

    // ---- Settings ----
    const settingsToggle = document.getElementById('settingsToggle');
    const settingsPanel = document.getElementById('settingsPanel');
    const toggleArrow = document.getElementById('toggleArrow');
    const apiKeyInput = document.getElementById('apiKeyInput');
    const baseUrlInput = document.getElementById('baseUrlInput');
    const modelInput = document.getElementById('modelInput');
    const lanIpInput = document.getElementById('lanIpInput');
    const saveConfigBtn = document.getElementById('saveConfigBtn');
    const configStatus = document.getElementById('configStatus');

    settingsToggle.addEventListener('click', () => {
        settingsPanel.classList.toggle('hidden');
        toggleArrow.classList.toggle('open');
        if (!settingsPanel.classList.contains('hidden')) loadConfig();
    });

    async function loadConfig() {
        if (!(await ensurePaired())) return;
        fetch(withAuth('/api/config'))
            .then(r => r.json())
            .then(data => {
                apiKeyInput.placeholder = data.apiKey || 'sk-...';
                baseUrlInput.placeholder = data.baseUrl || 'https://api.deepseek.com';
                modelInput.placeholder = data.model || 'deepseek-chat';
                lanIpInput.placeholder = data.lanIp || 'auto';
            })
            .catch(() => { });
    }

    saveConfigBtn.addEventListener('click', async () => {
        if (!(await ensurePaired())) return;
        const body = {};
        if (apiKeyInput.value.trim()) body.apiKey = apiKeyInput.value.trim();
        if (baseUrlInput.value.trim()) body.baseUrl = baseUrlInput.value.trim();
        if (modelInput.value.trim()) body.model = modelInput.value.trim();
        if (lanIpInput.value.trim()) body.lanIp = lanIpInput.value.trim();

        if (Object.keys(body).length === 0) {
            configStatus.textContent = t('settings.needOneField');
            configStatus.className = 'config-status error';
            return;
        }

        saveConfigBtn.textContent = t('settings.saving');
        fetch(withAuth('/api/config'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        })
            .then(r => r.json())
            .then(data => {
                if (data.ok) {
                    aiAvailable = data.aiAvailable;
                    updateModeButtons();
                    configStatus.textContent = data.aiAvailable ? t('settings.saveOkAiOn') : t('settings.saveOk');
                    configStatus.className = 'config-status success';
                    apiKeyInput.value = '';
                    baseUrlInput.value = '';
                    modelInput.value = '';
                    lanIpInput.value = '';
                    loadConfig();
                } else {
                    configStatus.textContent = t('settings.saveFailed');
                    configStatus.className = 'config-status error';
                }
            })
            .catch(() => {
                configStatus.textContent = t('settings.networkError');
                configStatus.className = 'config-status error';
            })
            .finally(() => {
                saveConfigBtn.textContent = t('settings.save');
            });
    });

    // ---- PWA ----
    const installBtn = document.getElementById('installBtn');
    let deferredPrompt = null;

    window.addEventListener('beforeinstallprompt', (e) => {
        e.preventDefault();
        deferredPrompt = e;
        installBtn.classList.remove('hidden');
    });

    installBtn.addEventListener('click', async () => {
        if (deferredPrompt) {
            deferredPrompt.prompt();
            const result = await deferredPrompt.userChoice;
            if (result.outcome === 'accepted') {
                installBtn.classList.add('hidden');
            }
            deferredPrompt = null;
        } else {
            alert(t('pwa.installHint'));
        }
    });

    window.addEventListener('appinstalled', () => {
        installBtn.classList.add('hidden');
        deferredPrompt = null;
    });

    const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent);
    const isStandalone = window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone;
    if (isIOS && !isStandalone) {
        installBtn.classList.remove('hidden');
    }

    // ---- Init ----
    initLanguage();
    initAuthToken();
    connectWebSocket();
    initRecognition();
    inputText.focus();
})();

