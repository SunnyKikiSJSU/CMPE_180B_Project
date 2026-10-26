import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Book Metaverse",
  description: "Discover, organize, and track your reading life.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
