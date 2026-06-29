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
    var status = evt.detail.xhr.status;
    if (status === 429) {
        var retryAfter = parseInt(evt.detail.xhr.getResponseHeader('Retry-After'), 10);
        var message = 'Too many requests. Please try again';
        if (retryAfter && !isNaN(retryAfter)) {
            if (retryAfter < 60) {
                message += ' in ' + retryAfter + ' second' + (retryAfter === 1 ? '' : 's') + '.';
            } else {
                var minutes = Math.ceil(retryAfter / 60);
                message += ' in ' + minutes + ' minute' + (minutes === 1 ? '' : 's') + '.';
            }
        } else {
            message += ' later.';
        }
        if (window.Toast) {
            window.Toast.error(message);
        }
        return;
    }
    if (window.Toast) {
        window.Toast.error('An error occurred. Please try again.');
    }
});

document.body.addEventListener('htmx:sendError', function(evt) {
    if (window.Toast) {
        window.Toast.error('Network error. Please check your connection and try again.');
    }
});
