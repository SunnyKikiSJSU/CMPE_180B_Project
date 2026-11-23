"use client";

import { useState } from "react";

/** A destructive action that asks once inline ("Remove? Yes / Cancel") before running. */
export function ConfirmButton({ label, confirmLabel, question, onConfirm, pending = false, className = "text-button danger" }: { label: string; confirmLabel: string; question: string; onConfirm: () => void; pending?: boolean; className?: string }) {
  const [asking, setAsking] = useState(false);
  if (!asking) return <button type="button" className={className} onClick={() => setAsking(true)} disabled={pending}>{pending ? "Working…" : label}</button>;
  return (
    <span className="confirm-inline" role="group" aria-label={question}>
      <span>{question}</span>
      <button type="button" className="text-button danger" onClick={() => { setAsking(false); onConfirm(); }}>{confirmLabel}</button>
      <button type="button" className="text-button" onClick={() => setAsking(false)}>Cancel</button>
    </span>
  );
}
