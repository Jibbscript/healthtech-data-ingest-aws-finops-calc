import { Area, AreaChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { currency } from '../format';
import type { CostBreakdown } from '../model/cost-model';

interface ProjectionChartProps {
  breakdown: CostBreakdown;
}

export function ProjectionChart({ breakdown }: ProjectionChartProps) {
  const data = breakdown.projections.map((point) => ({
    month: point.month,
    compute: point.inference + point.processor,
    storage: point.storageStandard + point.storageIa + point.storageGlacier,
    rds: point.rds,
    network: point.dataTransfer + point.sqs,
    ops: point.observability + point.fixed,
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
          <Area type="monotone" dataKey="compute" stackId="1" stroke="#2563eb" fill="#93c5fd" />
          <Area type="monotone" dataKey="storage" stackId="1" stroke="#059669" fill="#86efac" />
          <Area type="monotone" dataKey="rds" stackId="1" stroke="#7c3aed" fill="#c4b5fd" />
          <Area type="monotone" dataKey="network" stackId="1" stroke="#ea580c" fill="#fdba74" />
          <Area type="monotone" dataKey="ops" stackId="1" stroke="#475569" fill="#cbd5e1" />
        </AreaChart>
      </ResponsiveContainer>
    </section>
  );
}
