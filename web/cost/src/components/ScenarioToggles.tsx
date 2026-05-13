import { currency } from '../format';
import type { CostBreakdown, ScenarioOptions } from '../model/cost-model';

interface ScenarioTogglesProps {
  scenario: ScenarioOptions;
  onChange: (scenario: ScenarioOptions) => void;
  base: CostBreakdown;
  withoutTiering: CostBreakdown;
  withoutSpot: CostBreakdown;
  withoutCompression: CostBreakdown;
}

export function ScenarioToggles({ scenario, onChange, base, withoutTiering, withoutSpot, withoutCompression }: ScenarioTogglesProps) {
  const set = (key: keyof ScenarioOptions) => onChange({ ...scenario, [key]: !scenario[key] });
  const delta = (other: CostBreakdown) => other.monthlyCost - base.monthlyCost;

  return (
    <section className="card" aria-labelledby="scenario-heading">
      <h2 id="scenario-heading">What-if toggles</h2>
      <div className="toggles">
        <label className="check-field">
          <input type="checkbox" checked={scenario.tiering} onChange={() => set('tiering')} />
          <span>With S3 lifecycle tiering</span>
        </label>
        <label className="check-field">
          <input type="checkbox" checked={scenario.spot} onChange={() => set('spot')} />
          <span>With processor Fargate Spot</span>
        </label>
        <label className="check-field">
          <input type="checkbox" checked={scenario.compression} onChange={() => set('compression')} />
          <span>With ingest compression</span>
        </label>
      </div>
      <dl className="delta-grid" aria-label="Scenario deltas versus optimized baseline">
        <div>
          <dt>No tiering</dt>
          <dd>{currency(delta(withoutTiering))}/mo</dd>
        </div>
        <div>
          <dt>No Spot</dt>
          <dd>{currency(delta(withoutSpot))}/mo</dd>
        </div>
        <div>
          <dt>No compression</dt>
          <dd>{currency(delta(withoutCompression))}/mo</dd>
        </div>
      </dl>
    </section>
  );
}
