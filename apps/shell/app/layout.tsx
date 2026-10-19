import "./globals.css";
import { RootHtml } from "@eam/ui";

export const metadata = { title: "EAM" };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <RootHtml>{children}</RootHtml>;
}
