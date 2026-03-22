import Link from "next/link";

export default function DashboardPage() {
  return (
    <main style={{ maxWidth: 900, margin: "40px auto", padding: 24 }}>
      <h1 style={{ marginTop: 0 }}>dashboard</h1>
      <ul style={{ lineHeight: 1.8 }}>
        <li>
          <Link href="/graph">graph visualization</Link>
        </li>
      </ul>
      <p style={{ opacity: 0.75 }}>
        use the graph page with a user id or graph id that exists in your open-context backend.
      </p>
    </main>
  );
}
