import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { currency } from '../format';
import type { CostBreakdown as Breakdown } from '../model/cost-model';

interface CostBreakdownProps {
  breakdown: Breakdown;
}

const colors = {
  compute: '#2563eb',
  storage: '#059669',
  rds: '#7c3aed',
  network: '#ea580c',
  ops: '#475569',
};

export function CostBreakdown({ breakdown }: CostBreakdownProps) {
  const row = {
    name: 'Monthly',
    compute: breakdown.categories.inference + breakdown.categories.processor,
    storage: breakdown.categories.storageStandard + breakdown.categories.storageIa + breakdown.categories.storageGlacier,
    rds: breakdown.categories.rds,
    network: breakdown.categories.dataTransfer + breakdown.categories.sqs,
    ops: breakdown.categories.observability + breakdown.categories.fixed,
  };

  return (
    <section className="card" aria-labelledby="breakdown-heading">
      <h2 id="breakdown-heading">Monthly cost breakdown</h2>
      <ResponsiveContainer width="100%" height={260}>
        <BarChart data={[row]} margin={{ top: 20, right: 20, bottom: 10, left: 10 }}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis tickFormatter={(value) => currency(Number(value))} />
          <Tooltip formatter={(value, name) => [`${currency(Number(value))} (${((Number(value) / breakdown.monthlyCost) * 100).toFixed(1)}%)`, name]} />
          <Legend />
          <Bar dataKey="compute" stackId="cost" fill={colors.compute} />
          <Bar dataKey="storage" stackId="cost" fill={colors.storage} />
          <Bar dataKey="rds" stackId="cost" fill={colors.rds} />
          <Bar dataKey="network" stackId="cost" fill={colors.network} />
          <Bar dataKey="ops" stackId="cost" fill={colors.ops} />
        </BarChart>
      </ResponsiveContainer>
      <dl className="summary-list">
        <div>
          <dt>Total monthly</dt>
          <dd>{currency(breakdown.monthlyCost)}</dd>
        </div>
        <div>
          <dt>Daily ingest</dt>
          <dd>{breakdown.ingestGbPerDay.toFixed(1)} GB</dd>
        </div>
      </dl>
    </section>
  );
}
