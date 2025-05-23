const API = {
    baseURL: '/api',
    
    async login(data) {
        try {
            console.log('Sending login request:', data);
            const response = await fetch(this.baseURL + '/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                    'Accept': 'application/json',
                    'Referer': window.location.origin + '/portal'
                },
                body: new URLSearchParams(data)
            });
            
            const result = await response.json();
            if (!response.ok) {
                throw new Error(result.message || '登录失败');
            }
            return result;
        } catch (error) {
            throw error;
        }
    },
    
    async logout(data) {
        try {
            const response = await fetch(this.baseURL + '/logout', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                    'Accept': 'application/json',
                    'Referer': window.location.origin + '/portal'
                },
                body: new URLSearchParams(data)
            });
            
            const result = await response.json();
            if (!response.ok) {
                throw new Error(result.message || '登出失败');
            }
            return result;
        } catch (error) {
            throw error;
        }
    }
};