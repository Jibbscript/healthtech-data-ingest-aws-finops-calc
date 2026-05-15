import { DEFAULT_PRICING, type Pricing, type RdsInstanceClass } from '../model/cost-model';

export interface PricingLoadResult {
  pricing: Pricing;
  source: 'api' | 'fallback';
  errors: string[];
}

type PricingSlice = Partial<{
  fargate: Partial<Pricing['fargate']>;
  s3: Partial<Pricing['s3']>;
  sqs: Partial<Pricing['sqs']>;
  rds: Partial<Record<RdsInstanceClass, number>>;
  dataTransfer: Partial<Pricing['dataTransfer']>;
  observability: Partial<Pricing['observability']>;
  compressionRatio: number;
  fixedMonthly: number;
}>;

const API_BASE = import.meta.env.VITE_PRICING_API_BASE ?? 'http://localhost:9000';

const ENDPOINTS = [
  `${API_BASE}/v1/pricing/fargate?region=us-east-1`,
  `${API_BASE}/v1/pricing/s3?region=us-east-1&class=standard`,
  `${API_BASE}/v1/pricing/s3?region=us-east-1&class=ia`,
  `${API_BASE}/v1/pricing/s3?region=us-east-1&class=glacier`,
  `${API_BASE}/v1/pricing/sqs?region=us-east-1`,
  `${API_BASE}/v1/pricing/rds?region=us-east-1&instance=db.m6g.large`,
  `${API_BASE}/v1/pricing/data-transfer?region=us-east-1&direction=out-internet`,
] as const;

export async function loadPricing(fetcher: typeof fetch = fetch): Promise<PricingLoadResult> {
  const results = await Promise.allSettled(
    ENDPOINTS.map(async (endpoint) => {
      const response = await fetcher(endpoint, { headers: { Accept: 'application/json' } });
      if (!response.ok) {
        throw new Error(`${endpoint} returned ${response.status}`);
      }
      return normalizeApiResponse(await response.json());
    }),
  );

  const errors: string[] = [];
  const slices: PricingSlice[] = [];
  results.forEach((result, index) => {
    if (result.status === 'fulfilled') {
      slices.push(result.value);
    } else {
      errors.push(`${ENDPOINTS[index]}: ${result.reason instanceof Error ? result.reason.message : String(result.reason)}`);
    }
  });

  return {
    pricing: mergePricing(slices),
    source: errors.length > 0 ? 'fallback' : 'api',
    errors,
  };
}

function normalizeApiResponse(raw: unknown): PricingSlice {
  if (!raw || typeof raw !== 'object') return {};
  const obj = raw as Record<string, unknown>;
  const slice: PricingSlice = {};
  if (typeof obj.fargate_vcpu_per_hour === 'number' || typeof obj.fargate_gb_per_hour === 'number') {
    slice.fargate = {
      vcpuHour: typeof obj.fargate_vcpu_per_hour === 'number' ? obj.fargate_vcpu_per_hour : undefined,
      gbHour: typeof obj.fargate_gb_per_hour === 'number' ? obj.fargate_gb_per_hour : undefined,
    };
  }
  if (typeof obj.s3_gb_month === 'number') {
    const klass = String(obj.s3_class ?? 'standard');
    slice.s3 = {};
    if (klass === 'standard') slice.s3.standardGbMonth = obj.s3_gb_month;
    if (klass === 'ia') slice.s3.standardIaGbMonth = obj.s3_gb_month;
    if (klass === 'glacier') slice.s3.glacierIrGbMonth = obj.s3_gb_month;
  }
  if (typeof obj.sqs_request_per_million === 'number') slice.sqs = { requestPerMillion: obj.sqs_request_per_million };
  if (typeof obj.rds_instance_per_hour === 'number') slice.rds = { 'db.m6g.large': obj.rds_instance_per_hour };
  if (typeof obj.data_transfer_gb === 'number') slice.dataTransfer = { internetEgressPerGb: obj.data_transfer_gb };
  return slice;
}

function mergePricing(slices: PricingSlice[]): Pricing {
  return slices.reduce<Pricing>(
    (pricing, slice) => ({
      ...pricing,
      fargate: { ...pricing.fargate, ...slice.fargate },
      s3: { ...pricing.s3, ...slice.s3 },
      sqs: { ...pricing.sqs, ...slice.sqs },
      rds: { ...pricing.rds, ...slice.rds },
      dataTransfer: { ...pricing.dataTransfer, ...slice.dataTransfer },
      observability: { ...pricing.observability, ...slice.observability },
      compressionRatio: slice.compressionRatio ?? pricing.compressionRatio,
      fixedMonthly: slice.fixedMonthly ?? pricing.fixedMonthly,
    }),
    DEFAULT_PRICING,
  );
}
