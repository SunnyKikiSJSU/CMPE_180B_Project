export type ReadingStatus = "want_to_read" | "reading" | "completed" | "paused";

export interface Author { id: string; name: string; }
export interface Category { id: string; name: string; }
export interface Book { id: string; title: string; author: string; authors?: Author[]; categories?: Category[]; coverUrl?: string; }
export interface ReadingProgress { bookId: string; status: ReadingStatus; currentPage: number; totalPages: number; updatedAt: string; }
export interface Rating { bookId: string; userId: string; value: number; }
export interface Review { id: string; bookId: string; userId: string; body: string; rating?: number; createdAt: string; }
export interface Shelf { id: string; name: string; count: number; }
export interface ReadingReport { booksRead: number; pagesRead: number; favoriteCategories: Category[]; recentActivity: ReadingProgress[]; }
