import { useEffect, useState } from 'react';
import { DEFAULT_PRICING, type Pricing } from '../model/cost-model';
import { loadPricing } from './pricing';

interface PricingState {
  pricing: Pricing;
  loading: boolean;
  warning: string | null;
}

export function usePricing(): PricingState {
  const [state, setState] = useState<PricingState>({ pricing: DEFAULT_PRICING, loading: true, warning: null });

  useEffect(() => {
    let alive = true;
    // loadPricing resolves with a fallback result even when every endpoint
    // fails (it uses Promise.allSettled), so there is no rejection to catch.
    loadPricing().then((result) => {
      if (!alive) return;
      setState({
        pricing: result.pricing,
        loading: false,
        warning: pricingWarning(result.source, result.errors.length),
      });
    });

    return () => {
      alive = false;
    };
  }, []);

  return state;
}

function pricingWarning(source: 'api' | 'partial' | 'fallback', errorCount: number): string | null {
  if (source === 'api') return null;
  if (source === 'partial') {
    return `Using bundled pricing defaults for ${errorCount} pricing endpoint(s); live pricing loaded for the remaining services.`;
  }
  return `Using bundled pricing snapshot because ${errorCount} pricing endpoint(s) failed.`;
}
