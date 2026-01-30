// API Configuration
const API_BASE = 'http://localhost:8080/api';

// Состояние приложения
let currentUser = null;
let token = null;

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
    // Проверяем сохраненный токен
    token = localStorage.getItem('token');
    if (token) {
        loadCurrentUser();
    }

    // Модальные окна
    const loginModal = document.getElementById('login-modal');
    const registerModal = document.getElementById('register-modal');
    const loginBtn = document.getElementById('login-btn');
    const registerSchoolBtn = document.getElementById('register-school-btn');
    const closeLogin = document.getElementById('close-login');
    const closeRegister = document.getElementById('close-register');

    // Открытие модальных окон
    loginBtn.addEventListener('click', () => {
        loginModal.style.display = 'block';
    });

    registerSchoolBtn.addEventListener('click', () => {
        registerModal.style.display = 'block';
    });

    // Закрытие модальных окон
    closeLogin.addEventListener('click', () => {
        loginModal.style.display = 'none';
    });

    closeRegister.addEventListener('click', () => {
        registerModal.style.display = 'none';
    });

    window.addEventListener('click', (e) => {
        if (e.target === loginModal) {
            loginModal.style.display = 'none';
        }
        if (e.target === registerModal) {
            registerModal.style.display = 'none';
        }
    });

    // Формы
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('register-form').addEventListener('submit', handleRegister);
    document.getElementById('logout-btn').addEventListener('click', handleLogout);
});

// Загрузка текущего пользователя
async function loadCurrentUser() {
    try {
        const response = await fetch(`${API_BASE}/auth/me`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (response.ok) {
            const data = await response.json();
            currentUser = data.user;
            showUserSection();
        } else {
            localStorage.removeItem('token');
            token = null;
        }
    } catch (error) {
        console.error('Failed to load user:', error);
        localStorage.removeItem('token');
        token = null;
    }
}

// Обработка входа
async function handleLogin(e) {
    e.preventDefault();
    
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    const errorDiv = document.getElementById('login-error');

    try {
        const response = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, password })
        });

        const data = await response.json();

        if (response.ok) {
            token = data.token;
            currentUser = data.user;
            localStorage.setItem('token', token);
            document.getElementById('login-modal').style.display = 'none';
            showUserSection();
        } else {
            errorDiv.textContent = data.error || 'Ошибка входа';
            errorDiv.style.display = 'block';
        }
    } catch (error) {
        errorDiv.textContent = 'Ошибка соединения с сервером';
        errorDiv.style.display = 'block';
    }
}

// Обработка регистрации
async function handleRegister(e) {
    e.preventDefault();
    
    const errorDiv = document.getElementById('register-error');

    // Сначала создаем школу
    const schoolData = {
        name: document.getElementById('school-name').value,
        address: document.getElementById('school-address').value,
        phone: document.getElementById('school-phone').value,
        email: document.getElementById('school-email').value
    };

    try {
        // Создаем школу (временно без авторизации - нужно доработать)
        const schoolResponse = await fetch(`${API_BASE}/schools`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(schoolData)
        });

        if (!schoolResponse.ok) {
            const schoolError = await schoolResponse.json();
            errorDiv.textContent = schoolError.error || 'Ошибка создания школы';
            errorDiv.style.display = 'block';
            return;
        }

        const schoolResult = await schoolResponse.json();
        const schoolId = schoolResult.school.id;

        // Регистрируем администратора
        const adminData = {
            school_id: schoolId,
            username: document.getElementById('admin-username').value,
            email: document.getElementById('admin-email').value,
            password: document.getElementById('admin-password').value,
            role: 'admin',
            first_name: document.getElementById('admin-firstname').value,
            last_name: document.getElementById('admin-lastname').value,
            middle_name: document.getElementById('admin-middlename').value
        };

        const adminResponse = await fetch(`${API_BASE}/auth/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(adminData)
        });

        const adminResult = await adminResponse.json();

        if (adminResponse.ok) {
            token = adminResult.token;
            currentUser = adminResult.user;
            localStorage.setItem('token', token);
            document.getElementById('register-modal').style.display = 'none';
            showUserSection();
            alert('Школа успешно зарегистрирована!');
        } else {
            errorDiv.textContent = adminResult.error || 'Ошибка регистрации администратора';
            errorDiv.style.display = 'block';
        }
    } catch (error) {
        errorDiv.textContent = 'Ошибка соединения с сервером';
        errorDiv.style.display = 'block';
    }
}

// Выход
function handleLogout() {
    localStorage.removeItem('token');
    token = null;
    currentUser = null;
    showGuestSection();
}

// Показать секцию для авторизованных
function showUserSection() {
    document.getElementById('guest-section').style.display = 'none';
    document.getElementById('user-section').style.display = 'block';
    
    // Обновляем информацию о пользователе
    const fullName = [currentUser.last_name, currentUser.first_name, currentUser.middle_name]
        .filter(Boolean)
        .join(' ');
    
    document.getElementById('user-name').textContent = fullName || currentUser.username;
    document.getElementById('user-role').textContent = getRoleLabel(currentUser.role);
    
    // Автоматический редирект на дашборд после успешного входа
    setTimeout(() => {
        window.location.href = '/static/pages/dashboard.html';
    }, 500);
}

// Показать секцию для гостей
function showGuestSection() {
    document.getElementById('guest-section').style.display = 'block';
    document.getElementById('user-section').style.display = 'none';
}

// Получить название роли
function getRoleLabel(role) {
    const roles = {
        'admin': 'Администратор',
        'teacher': 'Учитель',
        'student': 'Ученик',
        'parent': 'Родитель',
        'starosta': 'Староста'
    };
    return roles[role] || role;
}
