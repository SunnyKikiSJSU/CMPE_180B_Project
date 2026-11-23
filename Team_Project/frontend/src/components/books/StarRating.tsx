"use client";

/**
 * Five stars as a radio group (arrow keys move, Space/Enter picks). `onChange`
 * receives 1-5. Used for the standalone rating and inside the review form.
 */
export function StarRating({ value, onChange, label, disabled = false }: { value: number | null; onChange: (value: number) => void; label: string; disabled?: boolean }) {
  // Buttons are never `disabled` while saving: that would drop keyboard focus.
  const pick = (n: number) => { if (!disabled) onChange(n); };
  const move = (group: HTMLElement, delta: number) => {
    if (disabled) return;
    const next = Math.min(5, Math.max(1, (value ?? 0) + delta));
    pick(next);
    group.querySelectorAll<HTMLButtonElement>("button")[next - 1]?.focus();
  };
  return (
    <div className="star-rating" role="radiogroup" aria-label={label} aria-disabled={disabled || undefined}
      onKeyDown={(e) => {
        if (e.key === "ArrowRight" || e.key === "ArrowUp") { e.preventDefault(); move(e.currentTarget, 1); }
        if (e.key === "ArrowLeft" || e.key === "ArrowDown") { e.preventDefault(); move(e.currentTarget, -1); }
      }}>
      {[1, 2, 3, 4, 5].map((n) => (
        <button key={n} type="button" role="radio" aria-checked={value === n} aria-label={`${n} of 5`}
          tabIndex={value === n || (!value && n === 1) ? 0 : -1}
          className={value && n <= value ? "is-on" : undefined} aria-disabled={disabled || undefined} onClick={() => pick(n)}>★</button>
      ))}
    </div>
  );
}
