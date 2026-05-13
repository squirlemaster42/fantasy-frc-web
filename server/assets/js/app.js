document.addEventListener('DOMContentLoaded', function() {
    // 404 page dynamic message
    var notFoundMsg = document.getElementById('not-found-msg');
    if (notFoundMsg) {
        notFoundMsg.textContent = "The page '" + window.location.href + "' doesn't exist";
    }
});

document.body.addEventListener('htmx:configRequest', function(evt) {
    // Auto-inject CSRF token from cookie into HTMX request headers
    var token = document.cookie.split('; ').find(function(row) {
        return row.startsWith('csrf_token=');
    });
    if (token) {
        evt.detail.headers['X-CSRF-Token'] = token.split('=')[1];
    }
});

document.body.addEventListener('htmx:responseError', function(evt) {
    // Simple global error toast fallback
    var toast = document.getElementById('global-error-toast');
    if (toast) {
        toast.textContent = 'An error occurred. Please try again.';
        toast.classList.remove('hidden');
        setTimeout(function() {
            toast.classList.add('hidden');
        }, 4000);
    }
});
