const form = document.querySelector('#matrix-form');
const operationInput = document.querySelector('#operation');
const matrixInput = document.querySelector('#matrix');
const submitButton = form.querySelector('button');
const status = document.querySelector('#status');
const results = document.querySelector('#results');
const matricesOutput = document.querySelector('#matrices-output');
const statisticsOutput = document.querySelector('#statistics-output');

const examples = {
  rotate: '[[1, 2, 3], [4, 5, 6]]',
  qr: '[[1, 0], [0, 1], [0, 0]]'
};

function showStatus(message, type = '') {
  status.textContent = message;
  status.className = `status ${type}`;
}

operationInput.addEventListener('change', () => {
  matrixInput.value = examples[operationInput.value];
});

form.addEventListener('submit', async (event) => {
  event.preventDefault();

  let matrix;
  try {
    matrix = JSON.parse(matrixInput.value);
  } catch {
    results.hidden = true;
    showStatus('La matriz debe tener un formato JSON válido.', 'error');
    return;
  }

  submitButton.disabled = true;
  showStatus('Procesando matriz...');

  try {
    const response = await fetch('/api/process', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ operation: operationInput.value, matrix })
    });
    const payload = await response.json();

    if (!response.ok) {
      throw new Error(payload.error ?? 'No fue posible procesar la matriz.');
    }

    matricesOutput.textContent = JSON.stringify(payload.transformedMatrices, null, 2);
    statisticsOutput.textContent = JSON.stringify(payload.statistics, null, 2);
    results.hidden = false;
    showStatus('Matriz procesada correctamente.', 'success');
  } catch (error) {
    results.hidden = true;
    showStatus(error.message, 'error');
  } finally {
    submitButton.disabled = false;
  }
});
