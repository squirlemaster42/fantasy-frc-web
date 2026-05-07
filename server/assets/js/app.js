document.addEventListener('DOMContentLoaded', function() {
    // 404 page dynamic message
    var notFoundMsg = document.getElementById('not-found-msg');
    if (notFoundMsg) {
        notFoundMsg.textContent = "The page '" + window.location.href + "' doesn't exist";
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
