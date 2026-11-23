/** A skeleton grid with a status message for screen readers. */
export function LoadingState({ message = "Loading…", cards = 8 }: { message?: string; cards?: number }) {
  return (
    <div className="loading-state" role="status" aria-live="polite">
      <p>{message}</p>
      <div className="book-grid" aria-hidden="true">
        {Array.from({ length: cards }, (_, i) => <div className="book-card skeleton" key={i}><div className="cover cover-card" /><div className="book-card-body"><i /><i /><i /></div></div>)}
      </div>
    </div>
  );
}
