import type { CostInputs, RdsInstanceClass } from '../model/cost-model';

type NumericKey = {
  [K in keyof CostInputs]: CostInputs[K] extends number ? K : never;
}[keyof CostInputs];

interface InputPanelProps {
  inputs: CostInputs;
  onChange: (inputs: CostInputs) => void;
}

const numberFields: Array<{ key: NumericKey; label: string; min: number; max: number; step: number; help: string }> = [
  { key: 'dau', label: 'DAU', min: 100, max: 1_000_000, step: 100, help: 'Daily active users' },
  { key: 'capturesPerUserPerDay', label: 'Captures/user/day', min: 1, max: 10, step: 1, help: 'Capture frequency per active user' },
  { key: 'avgPayloadMb', label: 'Avg payload size (MB)', min: 1, max: 20, step: 0.5, help: 'Stored payload before compression' },
  { key: 'inferenceDurationSeconds', label: 'Inference duration (s)', min: 1, max: 60, step: 1, help: 'Seconds per inference capture' },
  { key: 'inferenceFargateVcpu', label: 'Inference Fargate vCPU', min: 0.25, max: 16, step: 0.25, help: 'vCPU allocated to inference task' },
  { key: 'processorFargateVcpu', label: 'Processor Fargate vCPU', min: 0.25, max: 4, step: 0.25, help: 'vCPU allocated to async processor' },
  { key: 'spotDiscountPercent', label: 'Spot discount %', min: 0, max: 100, step: 5, help: 'Discount applied only to processor pool' },
  { key: 'standardToIaDays', label: 'Standard→IA cutover (days)', min: 0, max: 365, step: 1, help: 'Lifecycle transition from S3 Standard' },
  { key: 'iaToGlacierDays', label: 'IA→Glacier cutover (days)', min: 0, max: 365, step: 1, help: 'Lifecycle transition from Standard-IA' },
];

const rdsClasses: RdsInstanceClass[] = ['db.t4g.medium', 'db.m6g.large', 'db.m6g.xlarge'];

export function InputPanel({ inputs, onChange }: InputPanelProps) {
  const update = <K extends keyof CostInputs>(key: K, value: CostInputs[K]) => onChange({ ...inputs, [key]: value });

  return (
    <section className="card" aria-labelledby="input-panel-heading">
      <h2 id="input-panel-heading">Cost levers</h2>
      <div className="input-grid">
        {numberFields.map((field) => (
          <label className="field" key={field.key}>
            <span>{field.label}</span>
            <input
              aria-label={field.label}
              aria-describedby={`${field.key}-help`}
              type="number"
              min={field.min}
              max={field.max}
              step={field.step}
              value={inputs[field.key]}
              onChange={(event) => update(field.key, Number(event.target.value))}
            />
            <input
              aria-label={`${field.label} slider`}
              type="range"
              min={field.min}
              max={field.max}
              step={field.step}
              value={inputs[field.key]}
              onChange={(event) => update(field.key, Number(event.target.value))}
            />
            <small id={`${field.key}-help`}>{field.help}</small>
          </label>
        ))}

        <label className="field">
          <span>RDS instance class</span>
          <select value={inputs.rdsInstanceClass} onChange={(event) => update('rdsInstanceClass', event.target.value as RdsInstanceClass)}>
            {rdsClasses.map((klass) => (
              <option key={klass} value={klass}>
                {klass}
              </option>
            ))}
          </select>
        </label>

        <label className="check-field">
          <input type="checkbox" checked={inputs.s3TieringEnabled} onChange={(event) => update('s3TieringEnabled', event.target.checked)} />
          <span>S3 tiering enabled</span>
        </label>

        <label className="check-field">
          <input type="checkbox" checked={inputs.readReplica} onChange={(event) => update('readReplica', event.target.checked)} />
          <span>Read replica?</span>
        </label>
      </div>
    </section>
  );
}
