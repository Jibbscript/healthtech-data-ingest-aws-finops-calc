import { describe, expect, it } from 'vitest';
import { DEFAULT_INPUTS, DEFAULT_PRICING, computeMonthlyCost } from './cost-model';

describe('computeMonthlyCost', () => {
  it('keeps the clean 300 DAU default well below the $6 target', () => {
    const result = computeMonthlyCost(DEFAULT_INPUTS, DEFAULT_PRICING);

    expect(result.perUserMonthlyCost).toBeLessThan(6);
    expect(result.ingestGbPerDay).toBeCloseTo(0.8789, 3);
    expect(result.categories.storageGlacier).toBeGreaterThan(result.categories.storageStandard);
  });

  it('handles 100k DAU with tiering and compression without crossing the target', () => {
    const result = computeMonthlyCost({ ...DEFAULT_INPUTS, dau: 100_000, avgPayloadMb: 7 }, DEFAULT_PRICING);

    expect(result.monthlyCost).toBeGreaterThan(5_000);
    expect(result.perUserMonthlyCost).toBeLessThan(1);
    expect(result.categories.storageGlacier).toBeGreaterThan(0);
  });

  it('makes no-tiering storage materially more expensive at year two', () => {
    const tiered = computeMonthlyCost({ ...DEFAULT_INPUTS, dau: 100_000, avgPayloadMb: 7 }, DEFAULT_PRICING, {
      tiering: true,
      spot: true,
      compression: true,
    });
    const untiered = computeMonthlyCost({ ...DEFAULT_INPUTS, dau: 100_000, avgPayloadMb: 7 }, DEFAULT_PRICING, {
      tiering: false,
      spot: true,
      compression: true,
    });

    expect(untiered.categories.storageStandard).toBeGreaterThan(tiered.categories.storageStandard);
    expect(untiered.monthlyCost - tiered.monthlyCost).toBeGreaterThan(5_000);
  });

  it('applies processor Spot discount only when enabled', () => {
    const withSpot = computeMonthlyCost({ ...DEFAULT_INPUTS, dau: 100_000 }, DEFAULT_PRICING, {
      tiering: true,
      spot: true,
      compression: true,
    });
    const withoutSpot = computeMonthlyCost({ ...DEFAULT_INPUTS, dau: 100_000 }, DEFAULT_PRICING, {
      tiering: true,
      spot: false,
      compression: true,
    });

    expect(withoutSpot.categories.processor).toBeGreaterThan(withSpot.categories.processor);
    expect(withoutSpot.categories.inference).toBeCloseTo(withSpot.categories.inference, 6);
  });

  it('models compression as a downstream ingest and storage lever', () => {
    const compressed = computeMonthlyCost(DEFAULT_INPUTS, DEFAULT_PRICING, { tiering: true, spot: true, compression: true });
    const raw = computeMonthlyCost(DEFAULT_INPUTS, DEFAULT_PRICING, { tiering: true, spot: true, compression: false });

    const totalStorageCost = (r: typeof raw) =>
      r.categories.storageStandard + r.categories.storageIa + r.categories.storageGlacier;
    expect(raw.ingestGbPerDay).toBeCloseTo(compressed.ingestGbPerDay * DEFAULT_PRICING.compressionRatio, 6);
    expect(totalStorageCost(raw)).toBeGreaterThan(totalStorageCost(compressed));
  });

  it('clamps invalid boundary inputs into supported ranges', () => {
    const result = computeMonthlyCost(
      {
        ...DEFAULT_INPUTS,
        dau: 0,
        avgPayloadMb: 99,
        capturesPerUserPerDay: 99,
        spotDiscountPercent: 150,
      },
      DEFAULT_PRICING,
    );

    expect(result.inputs.dau).toBe(1);
    expect(result.inputs.avgPayloadMb).toBe(20);
    expect(result.inputs.capturesPerUserPerDay).toBe(10);
    expect(result.inputs.spotDiscountPercent).toBe(100);
  });
});
