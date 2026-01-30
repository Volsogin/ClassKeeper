// Динамическая навигация по ролям
const MENU_ITEMS = {
    admin: [
        { href: '/pages/dashboard.html', text: 'Главная', icon: '🏠' },
        { href: '/pages/users.html', text: 'Пользователи', icon: '👥' },
        { href: '/pages/subjects.html', text: 'Предметы', icon: '📚' },
        { href: '/pages/classes.html', text: 'Классы', icon: '🎓' },
        { href: '/pages/schedule.html', text: 'Расписание', icon: '📅' },
        { href: '/pages/grades.html', text: 'Оценки', icon: '📊' },
        { href: '/pages/attendance.html', text: 'Посещаемость', icon: '✅' },
        { href: '/pages/homework.html', text: 'Домашние задания', icon: '📝' },
        { href: '/pages/announcements.html', text: 'Объявления', icon: '📢' },
        { href: '/pages/analytics.html', text: 'Аналитика', icon: '📈' },
        { href: '/pages/parent-student.html', text: 'Родители-Дети', icon: '👨‍👩‍👧' }
    ],
    teacher: [
        { href: '/pages/dashboard.html', text: 'Главная', icon: '🏠' },
        { href: '/pages/subjects.html', text: 'Предметы', icon: '📚' },
        { href: '/pages/classes.html', text: 'Классы', icon: '🎓' },
        { href: '/pages/schedule.html', text: 'Расписание', icon: '📅' },
        { href: '/pages/grades.html', text: 'Оценки', icon: '📊' },
        { href: '/pages/attendance.html', text: 'Посещаемость', icon: '✅' },
        { href: '/pages/homework.html', text: 'Домашние задания', icon: '📝' },
        { href: '/pages/announcements.html', text: 'Объявления', icon: '📢' }
    ],
    student: [
        { href: '/pages/dashboard.html', text: 'Главная', icon: '🏠' },
        { href: '/pages/classes.html', text: 'Мой класс', icon: '🎓' },
        { href: '/pages/schedule.html', text: 'Расписание', icon: '📅' },
        { href: '/pages/grades.html', text: 'Оценки', icon: '📊' },
        { href: '/pages/homework.html', text: 'Домашние задания', icon: '📝' },
        { href: '/pages/announcements.html', text: 'Объявления', icon: '📢' }
    ],
    parent: [
        { href: '/pages/dashboard.html', text: 'Главная', icon: '🏠' },
        { href: '/pages/classes.html', text: 'Класс ребёнка', icon: '🎓' },
        { href: '/pages/schedule.html', text: 'Расписание', icon: '📅' },
        { href: '/pages/grades.html', text: 'Оценки', icon: '📊' },
        { href: '/pages/homework.html', text: 'Домашние задания', icon: '📝' },
        { href: '/pages/announcements.html', text: 'Объявления', icon: '📢' }
    ]
};

function renderNavbar(currentPage) {
    const token = localStorage.getItem('token');
    if (!token) return;

    // Получаем роль из токена
    const payload = JSON.parse(atob(token.split('.')[1]));
    const userRole = payload.role || 'student';

    // Получаем меню для роли
    const menuItems = MENU_ITEMS[userRole] || MENU_ITEMS.student;

    // Рендерим навигацию
    const navbarMenu = document.querySelector('.navbar-menu');
    if (navbarMenu) {
        navbarMenu.innerHTML = menuItems.map(item => {
            const isActive = window.location.pathname.includes(item.href) || 
                           (currentPage && item.href.includes(currentPage));
            return `<a href="${item.href}" class="${isActive ? 'active' : ''}">${item.text}</a>`;
        }).join('');
    }

    // Устанавливаем имя пользователя - ЗАГРУЖАЕМ С СЕРВЕРА
    loadUserName();
}

async function loadUserName() {
    const token = localStorage.getItem('token');
    const API_BASE = 'http://localhost:8080/api';
    
    try {
        const response = await fetch(`${API_BASE}/auth/me`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        if (response.ok) {
            const data = await response.json();
            const user = data.user;
            const userName = `${user.last_name || ''} ${user.first_name || ''}`.trim() || user.username;
            const userNameEl = document.getElementById('user-name');
            if (userNameEl) {
                userNameEl.textContent = userName;
            }
        }
    } catch (error) {
        console.error('Error loading user:', error);
    }
}

// Автоматически вызываем при загрузке
document.addEventListener('DOMContentLoaded', () => {
    renderNavbar();
});
