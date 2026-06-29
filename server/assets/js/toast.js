(function() {
    'use strict';

    function showToast(message, type, duration) {
        type = type || 'error';
        duration = duration || 5000;

        var container = document.getElementById('toast-container');
        if (!container) {
            console.warn('Toast container not found');
            return;
        }

        // Dispatch a custom event on window so Alpine's .window listener catches it
        window.dispatchEvent(new CustomEvent('show-toast', {
            detail: { message: message, type: type, duration: duration }
        }));
    }

    window.Toast = {
        show: showToast,
        error: function(message, duration) {
            showToast(message, 'error', duration);
        },
        success: function(message, duration) {
            showToast(message, 'success', duration);
        },
        warning: function(message, duration) {
            showToast(message, 'warning', duration);
        },
        info: function(message, duration) {
            showToast(message, 'info', duration);
        }
    };
})();
