import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { currency } from '../format';
import type { CostBreakdown as Breakdown } from '../model/cost-model';
import { CATEGORY_COLORS, aggregateCategories } from './categories';

interface CostBreakdownProps {
  breakdown: Breakdown;
}

export function CostBreakdown({ breakdown }: CostBreakdownProps) {
  const row = { name: 'Monthly', ...aggregateCategories(breakdown.categories) };

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
          <Bar dataKey="compute" stackId="cost" fill={CATEGORY_COLORS.compute} />
          <Bar dataKey="storage" stackId="cost" fill={CATEGORY_COLORS.storage} />
          <Bar dataKey="rds" stackId="cost" fill={CATEGORY_COLORS.rds} />
          <Bar dataKey="network" stackId="cost" fill={CATEGORY_COLORS.network} />
          <Bar dataKey="ops" stackId="cost" fill={CATEGORY_COLORS.ops} />
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
