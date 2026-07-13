// Password Modal Functions
function openPasswordModal() {
    document.getElementById('password-modal').classList.add('active');
}

function closePasswordModal() {
    document.getElementById('password-modal').classList.remove('active');
    document.getElementById('password-error').innerHTML = '';
}

// Copy to Clipboard
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('Copied to clipboard!');
    }).catch(err => {
        console.error('Failed to copy:', err);
        // Fallback for older browsers
        const textarea = document.createElement('textarea');
        textarea.value = text;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        showToast('Copied to clipboard!');
    });
}

// Toggle API Key Visibility
function toggleKey(elementId, fullKey) {
    const element = document.getElementById(elementId);
    if (element.classList.contains('key-masked')) {
        element.classList.remove('key-masked');
        element.textContent = fullKey;
    } else {
        element.classList.add('key-masked');
        element.textContent = fullKey;
    }
}

// Toggle mask visibility (alias for compatibility)
function toggleMask(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        element.classList.toggle('key-masked');
    }
}

// Simple Toast Notification
function showToast(message) {
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = message;
    toast.style.cssText = `
        position: fixed;
        bottom: 2rem;
        right: 2rem;
        background: #2c3e50;
        color: white;
        padding: 1rem 1.5rem;
        border-radius: 4px;
        box-shadow: 0 4px 6px rgba(0,0,0,0.2);
        z-index: 10000;
        animation: slideIn 0.3s ease-out;
    `;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.style.animation = 'slideOut 0.3s ease-in';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// Add CSS animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(400px);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(400px);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);

// Handle HTMX events for better UX
document.body.addEventListener('htmx:afterRequest', (event) => {
    const xhr = event.detail.xhr;

    // Handle successful password change
    if (event.detail.pathInfo.requestPath === '/change-password' && xhr.status === 200) {
        showToast('Password changed successfully!');
        closePasswordModal();
        // Reload to update must_change_password flag
        setTimeout(() => location.reload(), 1000);
    }

    // Handle successful domain add
    if (event.detail.pathInfo.requestPath === '/add-domain' && xhr.status === 200) {
        event.detail.elt.reset();
        showToast('Domain added successfully!');
        // Clear any error messages
        const errorDiv = document.getElementById('domain-error');
        if (errorDiv) errorDiv.innerHTML = '';
    }

    // Handle successful key regeneration
    if (event.detail.pathInfo.requestPath.includes('/regenerate-key') && xhr.status === 200) {
        showToast('API key regenerated successfully!');
    }
});

// Handle errors
document.body.addEventListener('htmx:responseError', (event) => {
    const xhr = event.detail.xhr;
    let errorMessage = 'An error occurred. Please try again.';

    try {
        const response = JSON.parse(xhr.responseText);
        errorMessage = response.error || response.message || errorMessage;
    } catch (e) {
        errorMessage = xhr.responseText || errorMessage;
    }

    // Special handling for domain add errors
    if (event.detail.pathInfo.requestPath === '/add-domain') {
        const errorDiv = document.getElementById('domain-error');
        if (errorDiv) {
            errorDiv.innerHTML = errorMessage;
            // Auto-clear error after 5 seconds
            setTimeout(() => {
                errorDiv.innerHTML = '';
            }, 5000);
        }
        return;
    }

    // Show error in target or as toast
    if (event.detail.target) {
        event.detail.target.innerHTML = `<div class="error-message">${errorMessage}</div>`;
    } else {
        showToast('Error: ' + errorMessage);
    }
});

// Close modal on outside click
document.addEventListener('click', (e) => {
    const modal = document.getElementById('password-modal');
    if (modal && e.target === modal) {
        closePasswordModal();
    }
});

// Close modal on Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closePasswordModal();
    }
});

// Dark Mode Toggle - Initialize immediately for theme application
(function () {
    // Get saved theme from localStorage or default to light
    const currentTheme = localStorage.getItem('theme') || 'light';

    // Apply theme immediately to prevent flash
    document.documentElement.setAttribute('data-theme', currentTheme);

    // Wait for DOM to be ready before attaching event listeners
    function initThemeToggle() {
        const themeToggle = document.getElementById('theme-toggle');
        const themeIcon = document.getElementById('theme-icon');

        // Update icon based on current theme (Material Icons)
        if (themeIcon) {
            themeIcon.textContent = currentTheme === 'dark' ? 'light_mode' : 'dark_mode';
        }

        // Add click handler for theme toggle button
        if (themeToggle) {
            themeToggle.addEventListener('click', function () {
                const html = document.documentElement;
                const currentTheme = html.getAttribute('data-theme');

                if (currentTheme === 'dark') {
                    // Switch to light mode
                    html.setAttribute('data-theme', 'light');
                    localStorage.setItem('theme', 'light');
                    if (themeIcon) themeIcon.textContent = 'dark_mode';
                } else {
                    // Switch to dark mode
                    html.setAttribute('data-theme', 'dark');
                    localStorage.setItem('theme', 'dark');
                    if (themeIcon) themeIcon.textContent = 'light_mode';
                }
            });
        }
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initThemeToggle);
    } else {
        initThemeToggle();
    }
})();
