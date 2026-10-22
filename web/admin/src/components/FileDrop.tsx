import { useState } from "react";

export interface FileDropProps {
  id: string;
  label: string;
  hint?: string;
  accept?: string;
  disabled?: boolean;
  onFile: (file: File) => void;
}

/**
 * Accessible drag-and-drop file picker. The native file input stays focusable;
 * the visible label is its styled target.
 */
export function FileDrop({ id, label, hint, accept, disabled, onFile }: FileDropProps) {
  const [dragging, setDragging] = useState(false);

  const handleFileList = (files: FileList | null) => {
    const file = files?.[0];
    if (file) onFile(file);
  };

  return (
    <fieldset
      className={["filedrop", dragging ? "is-dragging" : "", disabled ? "is-disabled" : ""]
        .filter(Boolean)
        .join(" ")}
      onDragOver={(event) => {
        if (disabled) return;
        event.preventDefault();
        setDragging(true);
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={(event) => {
        if (disabled) return;
        event.preventDefault();
        setDragging(false);
        handleFileList(event.dataTransfer.files);
      }}
    >
      <input
        id={id}
        className="filedrop-input"
        type="file"
        accept={accept}
        disabled={disabled}
        onChange={(event) => {
          handleFileList(event.currentTarget.files);
          event.currentTarget.value = "";
        }}
      />
      <label className="filedrop-label" htmlFor={id}>
        <span className="filedrop-title">{label}</span>
        <span className="filedrop-hint">
          {hint ?? "Drag and drop a file here, or choose one from your computer."}
        </span>
      </label>
    </fieldset>
  );
}
