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
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        try {
            document.execCommand('copy');
            showToast('Copied to clipboard!');
        } catch (copyErr) {
            console.error('Fallback copy failed:', copyErr);
            showToast('Failed to copy to clipboard');
        }
        document.body.removeChild(textarea);
    });
}

// Handle copy button clicks with data attributes (for dynamically loaded content)
document.addEventListener('click', function (e) {
    // Check if the clicked element or its parent is a copy button with data-email
    const copyBtn = e.target.closest('.copy-email-btn');
    if (copyBtn && copyBtn.dataset.email) {
        e.preventDefault();
        copyToClipboard(copyBtn.dataset.email);
    }
});

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

// Email Inbox Split-Pane Functionality with URL State Management
(function () {
    // Get URL parameter
    function getURLParam(name) {
        const params = new URLSearchParams(window.location.search);
        return params.get(name);
    }

    // Set URL parameter without reload
    function setURLParam(name, value) {
        const url = new URL(window.location);
        if (value) {
            url.searchParams.set(name, value);
        } else {
            url.searchParams.delete(name);
        }
        window.history.replaceState({}, '', url);
    }

    // Setup click handlers for email items
    function setupEmailClickHandlers() {
        // Handle email address row clicks (expand/collapse email list)
        document.querySelectorAll('.email-address-row').forEach(row => {
            if (row.dataset.clickHandler) return;
            row.dataset.clickHandler = 'true';

            row.addEventListener('click', function (e) {
                const targetId = this.getAttribute('hx-target');
                if (!targetId) return;

                const target = document.querySelector(targetId);
                const icon = this.querySelector('.expand-icon');

                // If content already loaded, just toggle visibility
                if (target && target.innerHTML.trim() !== '') {
                    // IMPORTANT: Prevent default FIRST to stop HTMX
                    e.preventDefault();
                    e.stopPropagation();

                    // Toggle the visible class
                    target.classList.toggle('visible');

                    // Update icon rotation based on new state
                    if (icon) {
                        icon.style.transform = target.classList.contains('visible')
                            ? 'rotate(180deg)'
                            : 'rotate(0deg)';
                    }

                    return; // Exit early
                }

                // First time - let HTMX load, rotate icon down
                if (icon) icon.style.transform = 'rotate(180deg)';
                // Don't preventDefault - let HTMX load the content
            }, true); // Use capture phase to run before HTMX
        });

        // Handle individual email clicks (load into right panel)
        document.querySelectorAll('.email-item').forEach(item => {
            if (item.dataset.clickHandler) return;
            item.dataset.clickHandler = 'true';

            item.addEventListener('click', function (e) {
                const emailHeader = this.querySelector('.email-header');
                if (!emailHeader) return;

                const emailURL = emailHeader.getAttribute('hx-get');
                if (!emailURL) return;

                // Remove selected state from all emails
                document.querySelectorAll('.email-item').forEach(el => {
                    el.classList.remove('selected');
                });

                // Add selected state to this email
                this.classList.add('selected');

                // Load email body into right panel
                fetch(emailURL, {
                    headers: {
                        'Accept': 'text/html'
                    }
                })
                    .then(response => response.text())
                    .then(html => {
                        const viewer = document.getElementById('email-body-viewer');
                        if (viewer) {
                            viewer.innerHTML = html;
                        }

                        // Update URL with email ID
                        const emailId = emailURL.split('/').pop();
                        setURLParam('email', emailURL.replace('/dashboard/email/', '').replace(/\//g, '-'));
                    })
                    .catch(err => {
                        console.error('Failed to load email:', err);
                        showToast('Failed to load email');
                    });

                e.stopPropagation();
                e.preventDefault();
            });
        });
    }

    // Auto-select email from URL on page load
    function autoSelectFromURL() {
        const emailParam = getURLParam('email');
        if (!emailParam) return;

        // Format: domain-username-id
        const parts = emailParam.split('-');
        if (parts.length < 3) return;

        const id = parts.pop();
        const username = parts.pop();
        const domain = parts.join('-');

        // Find and expand the address if needed
        const addressRow = Array.from(document.querySelectorAll('.email-address-row')).find(row => {
            const target = row.getAttribute('hx-target');
            return target && target.includes(domain) && target.includes(username);
        });

        if (addressRow) {
            const targetId = addressRow.getAttribute('hx-target');
            const target = document.querySelector(targetId);

            // Expand if not already expanded
            if (target && !target.classList.contains('visible')) {
                // Trigger HTMX load if needed
                if (!target.innerHTML.trim()) {
                    htmx.trigger(addressRow, 'click');
                } else {
                    target.classList.add('visible');
                    const icon = addressRow.querySelector('.expand-icon');
                    if (icon) icon.style.transform = 'rotate(180deg)';
                }

                // Wait a bit for emails to load, then select
                setTimeout(() => {
                    const emailURL = `/dashboard/email/${domain}/${username}/${id}`;
                    const emailItem = Array.from(document.querySelectorAll('.email-header')).find(header => {
                        return header.getAttribute('hx-get') === emailURL;
                    });

                    if (emailItem) {
                        emailItem.closest('.email-item').click();
                    }
                }, 500);
            } else if (target && target.classList.contains('visible')) {
                // Already expanded, just select the email
                const emailURL = `/dashboard/email/${domain}/${username}/${id}`;
                const emailItem = Array.from(document.querySelectorAll('.email-header')).find(header => {
                    return header.getAttribute('hx-get') === emailURL;
                });

                if (emailItem) {
                    emailItem.closest('.email-item').click();
                }
            }
        }
    }

    // Initialize
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function () {
            setupEmailClickHandlers();
            setTimeout(autoSelectFromURL, 100);
        });
    } else {
        setupEmailClickHandlers();
        setTimeout(autoSelectFromURL, 100);
    }

    // Re-setup after HTMX loads new content
    document.body.addEventListener('htmx:afterSwap', function (event) {
        setupEmailClickHandlers();

        // Auto-show newly loaded email lists
        if (event.target.classList.contains('email-list-container')) {
            event.target.classList.add('visible');
        }

        // Try to auto-select from URL after content loads
        if (event.target.id === 'email-addresses-list') {
            setTimeout(autoSelectFromURL, 100);
        }
    });
})();

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
