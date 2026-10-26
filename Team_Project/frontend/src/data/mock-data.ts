import type { Book, Shelf } from "@/types/domain";

export const featuredBooks: Book[] = [
  { id: "night-circus", title: "The Night Circus", author: "Erin Morgenstern", coverUrl: "https://images.unsplash.com/photo-1543002588-bfa74002ed7e?auto=format&fit=crop&w=900&q=85", categories: [{ id: "fantasy", name: "Fantasy" }] },
  { id: "secret-garden", title: "The Secret Garden", author: "Frances Hodgson Burnett", coverUrl: "https://images.unsplash.com/photo-1512820790803-83ca734da794?auto=format&fit=crop&w=900&q=85", categories: [{ id: "classic", name: "Classic" }] },
  { id: "pride-prejudice", title: "Pride & Prejudice", author: "Jane Austen", coverUrl: "https://images.unsplash.com/photo-1544947950-fa07a98d237f?auto=format&fit=crop&w=900&q=85", categories: [{ id: "romance", name: "Romance" }] },
];

export const shelves: Shelf[] = [
  { id: "quiet-mornings", name: "Quiet mornings", count: 12 },
  { id: "romance", name: "A little romance", count: 8 },
  { id: "big-ideas", name: "Big ideas", count: 16 },
];
