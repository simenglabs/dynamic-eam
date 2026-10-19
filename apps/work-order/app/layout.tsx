import "./globals.css";
import { AppLayout } from "@eam/ui";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <AppLayout zone="work-order">{children}</AppLayout>;
}
