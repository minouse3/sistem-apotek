document.addEventListener('click', (event) => {
  const trigger = event.target.closest('[data-print-page]');
  if (!trigger) return;
  window.print();
});
