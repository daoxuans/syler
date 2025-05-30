const Utils = {
    getQueryParam(param) {
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get(param);
    },

    showMessage(message, type = 'error') {
        const msgEl = document.getElementById('message');
        msgEl.textContent = message;
        msgEl.className = `message ${type}`;
        msgEl.classList.remove('hidden');

        setTimeout(() => {
            msgEl.classList.add('hidden');
        }, 3000);
    },

    // 切换登录/登出界面
    toggleAuthSection(isLoggedIn, userData) {
        const loginSection = document.getElementById('loginSection');
        const logoutSection = document.getElementById('logoutSection');

        if (isLoggedIn && userData) {
            loginSection.classList.add('hidden');
            logoutSection.classList.remove('hidden');

            // 更新用户信息显示
            document.getElementById('displayUsername').textContent = userData.username || '';
            document.getElementById('displayUserIP').textContent = userData.userip || '';
            document.getElementById('displayTimeout').textContent = userData.timeout || '';
        } else {
            logoutSection.classList.add('hidden');
            loginSection.classList.remove('hidden');
        }
    },

    // 检查登录状态
    checkLoginStatus() {
        const isLoggedIn = localStorage.getItem('isLoggedIn') === 'true';
        if (isLoggedIn) {
            const userData = {
                username: localStorage.getItem('username'),
                userip: localStorage.getItem('userip'),
                timeout: localStorage.getItem('timeout')
            };
            this.toggleAuthSection(true, userData);
        }
        return isLoggedIn;
    },

    // 保存登录信息
    saveLoginData(data) {
        localStorage.setItem('isLoggedIn', 'true');
        localStorage.setItem('username', data.username);
        localStorage.setItem('userip', data.userip);
        localStorage.setItem('timeout', data.timeout);
    },

    // 清除登录信息
    clearLoginData() {
        localStorage.removeItem('isLoggedIn');
        localStorage.removeItem('username');
        localStorage.removeItem('userip');
        localStorage.removeItem('timeout');
    },

    startLogoutTimer(timeout) {
        const ms = parseInt(timeout) * 1000;
        return setTimeout(() => {
            this.handleAutoLogout();
        }, ms);
    },

    async handleAutoLogout() {
        const nasip = localStorage.getItem('nasip');
        const userip = localStorage.getItem('userip');

        try {
            await API.logout({ nasip, userip });
            this.showMessage('会话已过期，请重新登录', 'info');
            setTimeout(() => {
                window.location.reload();
            }, 2000);
        } catch (error) {
            console.error('Auto logout failed:', error);
        }
    }
};