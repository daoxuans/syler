document.addEventListener('DOMContentLoaded', () => {
    // 初始化参数
    const nasip = Utils.getQueryParam('nasip');
    const userip = Utils.getQueryParam('userip');
    
    if (!nasip || !userip) {
        Utils.showMessage('缺少必要参数');
        return;
    }
    
    // 设置隐藏字段
    document.getElementById('nasip').value = nasip;
    document.getElementById('userip').value = userip;
    localStorage.setItem('nasip', nasip);
    localStorage.setItem('userip', userip);
    
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
    
    // 登出按钮处理
    const logoutButton = document.getElementById('logoutButton');
    
    logoutButton.addEventListener('click', async () => {
        try {
            const data = {
                nasip: localStorage.getItem('nasip'),
                userip: localStorage.getItem('userip')
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