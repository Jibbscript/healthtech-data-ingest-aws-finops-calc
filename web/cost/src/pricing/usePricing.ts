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
    loadPricing()
      .then((result) => {
        if (!alive) return;
        setState({
          pricing: result.pricing,
          loading: false,
          warning:
            result.source === 'fallback'
              ? `Using bundled pricing snapshot because ${result.errors.length} pricing endpoint(s) failed.`
              : null,
        });
      })
      .catch((error: unknown) => {
        if (!alive) return;
        setState({
          pricing: DEFAULT_PRICING,
          loading: false,
          warning: `Using bundled pricing snapshot because pricing could not be loaded: ${
            error instanceof Error ? error.message : String(error)
          }`,
        });
      });

    return () => {
      alive = false;
    };
  }, []);

  return state;
}
