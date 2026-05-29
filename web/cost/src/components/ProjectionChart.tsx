import { Area, AreaChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { currency } from '../format';
import type { CostBreakdown } from '../model/cost-model';
import { CATEGORY_COLORS, aggregateCategories } from './categories';

interface ProjectionChartProps {
  breakdown: CostBreakdown;
}

export function ProjectionChart({ breakdown }: ProjectionChartProps) {
  const data = breakdown.projections.map((point) => ({
    month: point.month,
    ...aggregateCategories(point),
    total: point.total,
  }));

  return (
    <section className="card" aria-labelledby="projection-heading">
      <h2 id="projection-heading">24-month projection</h2>
      <ResponsiveContainer width="100%" height={320}>
        <AreaChart data={data} margin={{ top: 20, right: 20, bottom: 10, left: 10 }}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="month" label={{ value: 'Month', position: 'insideBottom', offset: -5 }} />
          <YAxis tickFormatter={(value) => currency(Number(value))} />
          <Tooltip formatter={(value, name) => [currency(Number(value)), name]} labelFormatter={(label) => `Month ${label}`} />
          <Legend />
          <Area type="monotone" dataKey="compute" stackId="1" stroke={CATEGORY_COLORS.compute} fill="#93c5fd" />
          <Area type="monotone" dataKey="storage" stackId="1" stroke={CATEGORY_COLORS.storage} fill="#86efac" />
          <Area type="monotone" dataKey="rds" stackId="1" stroke={CATEGORY_COLORS.rds} fill="#c4b5fd" />
          <Area type="monotone" dataKey="network" stackId="1" stroke={CATEGORY_COLORS.network} fill="#fdba74" />
          <Area type="monotone" dataKey="ops" stackId="1" stroke={CATEGORY_COLORS.ops} fill="#cbd5e1" />
        </AreaChart>
      </ResponsiveContainer>
    </section>
  );
}
