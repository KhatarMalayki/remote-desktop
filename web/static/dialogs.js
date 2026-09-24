let appDialogQueue = Promise.resolve();

function appDialog(message, mode, initialValue = '', options = {}) {
  const task = appDialogQueue.then(() => new Promise(resolve => {
    const previousFocus = document.activeElement;
    const dialog = document.createElement('dialog');
    dialog.className = 'app-dialog';
    dialog.setAttribute('aria-labelledby', 'app-dialog-title');
    dialog.setAttribute('aria-describedby', 'app-dialog-message');
    dialog.innerHTML = '<form method="dialog"><h3 id="app-dialog-title"></h3><p id="app-dialog-message"></p><label for="app-dialog-input">Nilai</label><input id="app-dialog-input"><div class="modal-actions"><button type="button" class="btn btn-ghost" data-cancel>Batal</button><button type="submit" class="btn btn-primary" data-submit></button></div></form>';
    dialog.querySelector('h3').textContent = mode === 'confirm' ? 'Konfirmasi tindakan' : mode === 'prompt' ? 'Masukkan informasi' : 'Informasi';
    dialog.querySelector('p').textContent = String(message);
    const input = dialog.querySelector('input');
    input.hidden = mode !== 'prompt';
    dialog.querySelector('label').hidden = input.hidden;
    input.type = options.type || 'text';
    input.autocomplete = input.type === 'password' ? 'new-password' : 'off';
    input.value = initialValue;
    const cancel = dialog.querySelector('[data-cancel]');
    cancel.hidden = mode === 'alert';
    dialog.querySelector('[data-submit]').textContent = mode === 'confirm' ? 'Ya, lanjutkan' : mode === 'prompt' ? 'Simpan' : 'Tutup';
    let result = mode === 'prompt' ? null : false;
    cancel.addEventListener('click', () => dialog.close());
    dialog.querySelector('form').addEventListener('submit', event => {
      event.preventDefault();
      result = mode === 'prompt' ? input.value : true;
      dialog.close();
    });
    dialog.addEventListener('close', () => {
      input.value = '';
      dialog.remove();
      if (previousFocus && previousFocus.isConnected) previousFocus.focus();
      resolve(result);
    }, { once: true });
    document.body.appendChild(dialog);
    dialog.showModal();
    if (mode === 'prompt') { input.focus(); input.select(); }
    else if (mode === 'confirm') cancel.focus();
    else dialog.querySelector('[data-submit]').focus();
  }));
  appDialogQueue = task.catch(() => {});
  return task;
}

function appAlert(message) { return appDialog(message, 'alert'); }
function appConfirm(message) { return appDialog(message, 'confirm'); }
function appPrompt(message, initialValue = '', options = {}) { return appDialog(message, 'prompt', initialValue, options); }
