import { describe, expect, it, vi } from 'vitest';
import { DEFAULT_PRICING } from '../model/cost-model';
import { loadPricing } from './pricing';

describe('loadPricing', () => {
  it('merges successful API pricing slices into the default model', async () => {
    const fetcher = vi.fn(fetchResponse);

    const result = await loadPricing(fetcher as unknown as typeof fetch);

    expect(result.source).toBe('api');
    expect(result.errors).toEqual([]);
    expect(result.pricing.fargate.vcpuHour).toBe(0.05);
    expect(result.pricing.s3.standardGbMonth).toBe(0.03);
    expect(result.pricing.s3.standardIaGbMonth).toBe(0.02);
    expect(result.pricing.s3.glacierIrGbMonth).toBe(0.01);
    expect(result.pricing.sqs.requestPerMillion).toBe(0.5);
    expect(result.pricing.rds['db.m6g.large']).toBe(0.2);
    expect(result.pricing.dataTransfer.internetEgressPerGb).toBe(0.1);
  });

  it('labels partial live pricing when only some endpoints fail', async () => {
    const fetcher = vi.fn(async (endpoint: string) => {
      if (endpoint.includes('/sqs')) {
        return { ok: false, status: 503, json: async () => ({}) } as Response;
      }
      return fetchResponse(endpoint);
    });

    const result = await loadPricing(fetcher as unknown as typeof fetch);

    expect(result.source).toBe('partial');
    expect(result.errors).toHaveLength(1);
    expect(result.pricing.sqs.requestPerMillion).toBe(DEFAULT_PRICING.sqs.requestPerMillion);
    expect(result.pricing.fargate.vcpuHour).toBe(0.05);
  });

  it('uses fallback only when every pricing endpoint fails', async () => {
    const fetcher = vi.fn(async () => ({ ok: false, status: 503, json: async () => ({}) })) as unknown as typeof fetch;

    const result = await loadPricing(fetcher);

    expect(result.source).toBe('fallback');
    expect(result.errors).toHaveLength(7);
    expect(result.pricing).toEqual(DEFAULT_PRICING);
  });

  it('ignores internal-shaped fields in API responses', async () => {
    const fetcher = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        fixedMonthly: 1,
        compressionRatio: 99,
        fargate: { vcpuHour: 99 },
        fargate_vcpu_per_hour: 0.05,
      }),
    })) as unknown as typeof fetch;

    const result = await loadPricing(fetcher);

    expect(result.pricing.fixedMonthly).toBe(DEFAULT_PRICING.fixedMonthly);
    expect(result.pricing.compressionRatio).toBe(DEFAULT_PRICING.compressionRatio);
    expect(result.pricing.fargate.vcpuHour).toBe(0.05);
  });
});

async function fetchResponse(endpoint: string) {
  const body = endpointBody(endpoint);
  return {
    ok: true,
    status: 200,
    json: async () => body,
  } as Response;
}

function endpointBody(endpoint: string) {
  if (endpoint.includes('/fargate')) return { fargate_vcpu_per_hour: 0.05, fargate_gb_per_hour: 0.006 };
  if (endpoint.includes('class=standard')) return { s3_class: 'standard', s3_gb_month: 0.03 };
  if (endpoint.includes('class=ia')) return { s3_class: 'ia', s3_gb_month: 0.02 };
  if (endpoint.includes('class=glacier')) return { s3_class: 'glacier', s3_gb_month: 0.01 };
  if (endpoint.includes('/sqs')) return { sqs_request_per_million: 0.5 };
  if (endpoint.includes('/rds')) return { rds_instance_per_hour: 0.2 };
  if (endpoint.includes('/data-transfer')) return { data_transfer_gb: 0.1 };
  return {};
}
