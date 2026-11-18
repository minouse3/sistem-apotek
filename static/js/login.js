const password = document.getElementById('password');
const toggle = document.getElementById('password-toggle');
const showIcon = toggle.querySelector('[data-show-icon]');
const hideIcon = toggle.querySelector('[data-hide-icon]');

toggle.addEventListener('click', () => {
  const visible = password.type === 'password';
  password.type = visible ? 'text' : 'password';
  showIcon.hidden = visible;
  hideIcon.hidden = !visible;
  toggle.setAttribute('aria-label', visible ? 'Sembunyikan password' : 'Tampilkan password');
});

toggle.hidden = false;

for (const id of ['username', 'password']) {
  const field = document.getElementById(id);
  const warning = document.getElementById(`${id}-required`);

  field.addEventListener('invalid', (event) => {
    event.preventDefault();
    warning.hidden = false;
    field.classList.add('login-required-invalid');
    field.setAttribute('aria-invalid', 'true');
    field.setAttribute('aria-describedby', warning.id);
    if (field === field.form.querySelector('input:invalid')) {
      field.focus();
    }
  });

  field.addEventListener('input', () => {
    if (warning.hidden) return;
    warning.hidden = true;
    field.classList.remove('login-required-invalid');
    field.removeAttribute('aria-invalid');
    field.removeAttribute('aria-describedby');
  });
}

const credentialError = document.querySelector('[data-credential-error]');
if (credentialError) {
  const form = credentialError.closest('form');
  const invalidFields = form.querySelectorAll('input[aria-describedby="login-error"]');

  form.addEventListener('input', () => {
    credentialError.remove();
    for (const field of invalidFields) {
      if (field.classList.contains('login-required-invalid')) continue;
      field.removeAttribute('aria-invalid');
      field.removeAttribute('aria-describedby');
    }
  }, { once: true });
}
