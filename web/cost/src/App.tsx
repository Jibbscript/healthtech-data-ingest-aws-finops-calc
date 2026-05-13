import { useMemo, useState } from 'react';
import { CostBreakdown } from './components/CostBreakdown';
import { ExportButton } from './components/ExportButton';
import { InputPanel } from './components/InputPanel';
import { PerUserDisplay } from './components/PerUserDisplay';
import { ProjectionChart } from './components/ProjectionChart';
import { ScenarioToggles } from './components/ScenarioToggles';
import { currency } from './format';
import { DEFAULT_INPUTS, DEFAULT_SCENARIO, computeMonthlyCost, computeScenarioDeltas, type CostInputs, type ScenarioOptions } from './model/cost-model';
import { usePricing } from './pricing/usePricing';

export function App() {
  const [inputs, setInputs] = useState<CostInputs>(DEFAULT_INPUTS);
  const [scenario, setScenario] = useState<ScenarioOptions>(DEFAULT_SCENARIO);
  const { pricing, loading, warning } = usePricing();

  const breakdown = useMemo(() => computeMonthlyCost(inputs, pricing, scenario), [inputs, pricing, scenario]);
  const deltas = useMemo(() => computeScenarioDeltas(inputs, pricing), [inputs, pricing]);

  return (
    <main className="app-shell">
      <header className="hero">
        <div>
          <p className="eyebrow">Throne PoC #1</p>
          <h1>Healthtech ingest cost calculator</h1>
          <p>Change any lever to recompute monthly AWS cost, unit economics, and 24-month storage accumulation.</p>
        </div>
        <ExportButton targetId="cost-dashboard" />
      </header>

      {loading ? <div role="status" className="notice">Loading live pricing…</div> : null}
      {warning ? <div role="alert" className="notice warning">{warning}</div> : null}

      <div id="cost-dashboard" className="dashboard">
        <InputPanel inputs={inputs} onChange={setInputs} />
        <div className="metric-column">
          <PerUserDisplay value={breakdown.perUserMonthlyCost} />
          <section className="card" aria-labelledby="total-heading">
            <h2 id="total-heading">Total monthly estimate</h2>
            <p className="total-cost">{currency(breakdown.monthlyCost)}</p>
            <p>{breakdown.ingestGbPerDay.toFixed(1)} GB/day after {scenario.compression ? '5× compression' : 'no compression'}.</p>
          </section>
          <ScenarioToggles
            scenario={scenario}
            onChange={setScenario}
            base={deltas.base}
            withoutTiering={deltas.withoutTiering}
            withoutSpot={deltas.withoutSpot}
            withoutCompression={deltas.withoutCompression}
          />
        </div>
        <CostBreakdown breakdown={breakdown} />
        <ProjectionChart breakdown={breakdown} />
      </div>
    </main>
  );
}
