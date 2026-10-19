import { LoginForm } from "./LoginForm";

export const metadata = { title: "Masuk · EAM" };

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ next?: string }> }) {
  const { next } = await searchParams;
  return (
    <div className="flex min-h-screen items-center justify-center bg-[var(--background)] p-4">
      <LoginForm next={next ?? ""} />
    </div>
  );
}
