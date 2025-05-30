document.addEventListener('DOMContentLoaded', () => {
    // 初始化参数
    const nasip = Utils.getQueryParam('nasip');
    const userip = Utils.getQueryParam('userip');
    const usermac = Utils.getQueryParam('usermac');

    if (!nasip || !userip || !usermac) {
        Utils.showMessage('缺少必要参数');
        return;
    }

    // 设置隐藏字段
    document.getElementById('nasip').value = nasip;
    document.getElementById('userip').value = userip;
    document.getElementById('usermac').value = usermac;
    localStorage.setItem('nasip', nasip);
    localStorage.setItem('userip', userip);
    localStorage.setItem('usermac', usermac);

    // 检查登录状态
    Utils.checkLoginStatus();

    // 登录表单提交处理
    const loginForm = document.getElementById('loginForm');
    const loginButton = document.getElementById('loginButton');

    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        loginButton.disabled = true;

        try {
            const formData = new FormData(loginForm);
            const data = Object.fromEntries(formData.entries());
            console.log('Form data to be sent:', data);

            const result = await API.login(data);
            Utils.showMessage(result.message, 'success');

            // 保存登录信息并更新界面
            Utils.saveLoginData(result.data);
            Utils.toggleAuthSection(true, result.data);

            // 更新浏览器历史状态，使移动设备显示"完成"而不是"取消"
            window.history.pushState({ authenticated: true }, '', window.location.href);
            window.history.go(0);

        } catch (error) {
            Utils.showMessage(error.message);
        } finally {
            loginButton.disabled = false;
        }
    });

    // 获取验证码按钮处理
    const getCodeButton = document.getElementById('getCodeButton');
    let countdown = 0;

    getCodeButton.addEventListener('click', async () => {
        if (countdown > 0) {
            Utils.showMessage(`请等待 ${countdown} 秒后重试`, 'info');
            return;
        }

        const usernameInput = document.getElementById('username');
        const phone = usernameInput.value.trim();

        if (!phone) {
            Utils.showMessage('请输入用户名作为手机号', 'error');
            return;
        }

        // 启动倒计时
        countdown = 60;
        const interval = setInterval(() => {
            if (countdown <= 0) {
                clearInterval(interval);
                getCodeButton.textContent = "获取密码";
            } else {
                getCodeButton.textContent = `${countdown--} 秒`;
            }
        }, 1000);

        try {
            const response = await fetch('/api/sendcode', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Referer': window.location.origin + '/portal'
                },
                body: JSON.stringify({ phone })
            });

            if (response.ok) {
                Utils.showMessage('验证码发送成功，请注意查收，5分钟有效', 'success');
            } else {
                const result = await response.json();
                Utils.showMessage(result.message || '获取失败，请稍后再试', 'error');
            }
        } catch (error) {
            Utils.showMessage('网络错误，请检查连接', 'error');
        }
    });

    // 登出按钮处理
    const logoutButton = document.getElementById('logoutButton');

    logoutButton.addEventListener('click', async () => {
        try {
            const data = {
                nasip: localStorage.getItem('nasip'),
                userip: localStorage.getItem('userip'),
                usermac: localStorage.getItem('usermac')
            };

            const result = await API.logout(data);
            Utils.showMessage(result.message, 'success');

            // 清除登录信息并更新界面
            Utils.clearLoginData();
            Utils.toggleAuthSection(false);

        } catch (error) {
            Utils.showMessage(error.message);
        }
    });
});