"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    if (!res.ok) {
      setError("invalid credentials");
      return;
    }
    router.push("/dashboard");
  }

  return (
    <main style={{ maxWidth: 420, margin: "80px auto", padding: 24 }}>
      <h1 style={{ marginTop: 0 }}>open-context</h1>
      <p style={{ opacity: 0.8 }}>admin sign-in</p>
      <form onSubmit={onSubmit} style={{ display: "grid", gap: 12 }}>
        <input
          placeholder="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          style={{ padding: 10, borderRadius: 8, border: "1px solid #30363d", background: "#0d1117", color: "#e6edf3" }}
        />
        <input
          placeholder="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          style={{ padding: 10, borderRadius: 8, border: "1px solid #30363d", background: "#0d1117", color: "#e6edf3" }}
        />
        <button type="submit" style={{ padding: 10, borderRadius: 8, border: 0, background: "#238636", color: "white", fontWeight: 600 }}>
          sign in
        </button>
        {error ? <div style={{ color: "#f85149" }}>{error}</div> : null}
      </form>
    </main>
  );
}
