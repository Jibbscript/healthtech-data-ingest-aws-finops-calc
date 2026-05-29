import type { CostCategoryBreakdown } from '../model/cost-model';

export interface CategoryTotals {
  compute: number;
  storage: number;
  rds: number;
  network: number;
  ops: number;
}

// aggregateCategories collapses the ten raw cost categories into the five
// display buckets both charts render, so the bucket math lives in one place.
export function aggregateCategories(c: CostCategoryBreakdown): CategoryTotals {
  return {
    compute: c.inference + c.processor,
    storage: c.storageStandard + c.storageIa + c.storageGlacier,
    rds: c.rds,
    network: c.dataTransfer + c.sqs,
    ops: c.observability + c.fixed,
  };
}

export const CATEGORY_COLORS: Record<keyof CategoryTotals, string> = {
  compute: '#2563eb',
  storage: '#059669',
  rds: '#7c3aed',
  network: '#ea580c',
  ops: '#475569',
};
