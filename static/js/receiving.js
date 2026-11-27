(() => {
  const lineList = document.querySelector("[data-receiving-lines]");
  const template = document.getElementById("receiving-line-template");
  const addButton = document.querySelector("[data-add-line]");

  if (!(lineList instanceof HTMLElement) || !(template instanceof HTMLTemplateElement) ||
      !(addButton instanceof HTMLButtonElement)) {
    return;
  }

  const maximumLines = 50;
  let nextId = lineList.querySelectorAll("[data-receiving-line]").length;

  const updateLineControls = () => {
    const lines = Array.from(lineList.querySelectorAll("[data-receiving-line]"));
    lines.forEach((line, index) => {
      const legend = line.querySelector("[data-line-label]");
      const removeButton = line.querySelector("[data-remove-line]");
      if (legend instanceof HTMLElement) {
        legend.textContent = `Obat ${index + 1}`;
      }
      if (removeButton instanceof HTMLButtonElement) {
        removeButton.hidden = lines.length <= 1;
        removeButton.setAttribute("aria-label", `Hapus baris obat ${index + 1}`);
      }
    });
    addButton.disabled = lines.length >= maximumLines;
    addButton.hidden = false;
  };

  addButton.addEventListener("click", () => {
    if (lineList.querySelectorAll("[data-receiving-line]").length >= maximumLines) {
      return;
    }

    const line = document.importNode(template.content, true).querySelector("[data-receiving-line]");
    if (!(line instanceof HTMLFieldSetElement)) {
      return;
    }

    const lineId = nextId++;
    line.querySelectorAll("[data-field]").forEach((field) => {
      if (!(field instanceof HTMLElement)) {
        return;
      }
      const key = field.getAttribute("data-field");
      if (!key) {
        return;
      }
      const id = `receiving-${key}-${lineId}`;
      field.id = id;
      const label = line.querySelector(`[data-label-for="${key}"]`);
      if (label instanceof HTMLLabelElement) {
        label.htmlFor = id;
      }
    });

    lineList.append(line);
    updateLineControls();
    const firstField = line.querySelector("select");
    if (firstField instanceof HTMLSelectElement) {
      firstField.focus();
    }
  });

  lineList.addEventListener("click", (event) => {
    if (!(event.target instanceof Element)) {
      return;
    }
    const removeButton = event.target.closest("[data-remove-line]");
    const line = removeButton?.closest("[data-receiving-line]");
    if (!line || lineList.querySelectorAll("[data-receiving-line]").length <= 1) {
      return;
    }
    line.remove();
    updateLineControls();
    addButton.focus();
  });

  updateLineControls();
})();
