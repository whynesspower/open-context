import { ArrowRight, Network, ShieldCheck, Waypoints } from 'lucide-react';
import Link from 'next/link';

import { AppShell } from '@/components/app-shell';
import { Card } from '@/components/card';

const workflow = [
  'Authenticate into the protected admin workspace.',
  'Open the graph explorer and load a user or group graph id.',
  'Inspect relationships visually and confirm the returned triplets underneath the canvas.',
];

export default function DashboardPage() {
  return (
    <AppShell
      title="Admin dashboard"
      description="A clean operational surface for loading graph data, validating relationships, and working from a protected admin session."
      actions={
        <Link className="button button-primary button-md" href="/graph">
          Open graph explorer
          <ArrowRight size={16} />
        </Link>
      }
    >
      <section className="dashboard-grid">
        <Card
          description="Use the graph explorer with a valid user UUID or graph UUID that exists in your Open Context backend."
          icon={<Network size={18} />}
          title="Explore graph relationships"
        >
          <div className="stack">
            {workflow.map((step, index) => (
              <div className="dashboard-callout" key={step}>
                <span className="kicker">Step 0{index + 1}</span>
                <h3>{step}</h3>
              </div>
            ))}
            <div className="link-row">
              <Link className="button button-secondary button-md" href="/graph">
                Launch graph explorer
              </Link>
            </div>
          </div>
        </Card>

        <div className="stack">
          <Card
            description="The UI now uses a shared shell, reusable controls, and a darker visual system tuned for internal operations."
            icon={<Waypoints size={18} />}
            title="What changed"
          >
            <div className="stack">
              <div className="dashboard-callout">
                <h3>Consistent experience</h3>
                <p className="muted">
                  Shared navigation, cards, inputs, actions, and page headers replace duplicated
                  inline styling.
                </p>
              </div>
              <div className="dashboard-callout">
                <h3>Faster graph access</h3>
                <p className="muted">
                  The graph route is now positioned as the primary admin workflow instead of a bare
                  link in a list.
                </p>
              </div>
            </div>
          </Card>

          <Card
            description="The admin session is now protected by signed cookies rather than a client-forgeable flag."
            icon={<ShieldCheck size={18} />}
            title="Security posture"
          >
            <dl className="meta-list">
              <div>
                <dt>Cookie</dt>
                <dd>Signed and expiring</dd>
              </div>
              <div>
                <dt>Route protection</dt>
                <dd>Shared verification in middleware and API handlers</dd>
              </div>
              <div>
                <dt>Login hardening</dt>
                <dd>Basic throttling for repeated failed attempts</dd>
              </div>
            </dl>
          </Card>
        </div>
      </section>
    </AppShell>
  );
}
