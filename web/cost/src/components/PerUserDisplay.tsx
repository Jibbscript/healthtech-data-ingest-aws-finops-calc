import { currency } from '../format';

interface PerUserDisplayProps {
  value: number;
}

function perUserBand(value: number) {
  if (value <= 6) return 'green';
  if (value <= 8) return 'amber';
  return 'red';
}

export function PerUserDisplay({ value }: PerUserDisplayProps) {
  const band = perUserBand(value);
  return (
    <section className={`per-user per-user-${band}`} aria-label="Per user monthly cost">
      <span className="eyebrow">$/user/month</span>
      <strong>{currency(value, 2)}</strong>
      <span>{band === 'green' ? 'Inside the $6 target' : band === 'amber' ? 'Near the $6 target' : 'Above the $8 risk line'}</span>
    </section>
  );
}
