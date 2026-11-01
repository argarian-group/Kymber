import { type ReactNode, useEffect, useRef } from "react";

type ModalProps = {
  label: string;
  className: string;
  onClose: () => void;
  onCancel?: () => void;
  children: ReactNode;
};

export function Modal({ label, className, onClose, onCancel, children }: ModalProps) {
  const dialog = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    dialog.current?.showModal();
  }, []);

  return (
    <dialog ref={dialog} className={`m-auto w-[calc(100%-2rem)] rounded-panel bg-raised p-0 text-ink backdrop:bg-black/60 ${className}`} aria-label={label} onCancel={onCancel} onClose={onClose}>
      {children}
    </dialog>
  );
}
