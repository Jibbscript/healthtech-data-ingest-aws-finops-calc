export type RdsInstanceClass = 'db.t4g.medium' | 'db.m6g.large' | 'db.m6g.xlarge';

export interface CostInputs {
  dau: number;
  capturesPerUserPerDay: number;
  avgPayloadMb: number;
  inferenceDurationSeconds: number;
  inferenceFargateVcpu: number;
  processorFargateVcpu: number;
  spotDiscountPercent: number;
  s3TieringEnabled: boolean;
  standardToIaDays: number;
  iaToGlacierDays: number;
  rdsInstanceClass: RdsInstanceClass;
  readReplica: boolean;
}

export interface ScenarioOptions {
  tiering: boolean;
  spot: boolean;
  compression: boolean;
}

export interface Pricing {
  fargate: {
    vcpuHour: number;
    gbHour: number;
    armDiscountPercent: number;
    inferenceMemoryGbPerVcpu: number;
    processorMemoryGbPerVcpu: number;
    processorDurationSeconds: number;
  };
  s3: {
    standardGbMonth: number;
    standardIaGbMonth: number;
    glacierIrGbMonth: number;
    putPer1000: number;
  };
  sqs: { requestPerMillion: number };
  rds: Record<RdsInstanceClass, number>;
  dataTransfer: { internetEgressPerGb: number; egressGbPerCapture: number };
  observability: { logsPerGb: number; logMbPerUserMonth: number; metricsFixedMonthly: number };
  compressionRatio: number;
  fixedMonthly: number;
}

export interface CostCategoryBreakdown {
  inference: number;
  processor: number;
  storageStandard: number;
  storageIa: number;
  storageGlacier: number;
  dataTransfer: number;
  rds: number;
  sqs: number;
  observability: number;
  fixed: number;
}

export interface StorageBreakdown {
  standardGb: number;
  iaGb: number;
  glacierGb: number;
}

export interface MonthlyProjectionPoint extends CostCategoryBreakdown {
  month: number;
  total: number;
}

export interface CostBreakdown {
  inputs: CostInputs;
  scenario: ScenarioOptions;
  monthlyCost: number;
  perUserMonthlyCost: number;
  ingestGbPerDay: number;
  categories: CostCategoryBreakdown;
  projections: MonthlyProjectionPoint[];
}

export const DEFAULT_INPUTS: CostInputs = {
  dau: 300,
  capturesPerUserPerDay: 3,
  avgPayloadMb: 5,
  inferenceDurationSeconds: 15,
  inferenceFargateVcpu: 2,
  processorFargateVcpu: 1,
  spotDiscountPercent: 70,
  s3TieringEnabled: true,
  standardToIaDays: 30,
  iaToGlacierDays: 90,
  rdsInstanceClass: 'db.m6g.large',
  readReplica: false,
};

export const DEFAULT_SCENARIO: ScenarioOptions = {
  tiering: true,
  spot: true,
  compression: true,
};

export const DEFAULT_PRICING: Pricing = {
  fargate: {
    vcpuHour: 0.04048,
    gbHour: 0.004445,
    armDiscountPercent: 20,
    inferenceMemoryGbPerVcpu: 2,
    processorMemoryGbPerVcpu: 2,
    processorDurationSeconds: 2,
  },
  s3: {
    standardGbMonth: 0.023,
    standardIaGbMonth: 0.0125,
    glacierIrGbMonth: 0.004,
    putPer1000: 0.005,
  },
  sqs: { requestPerMillion: 0.4 },
  rds: {
    'db.t4g.medium': 0.067,
    'db.m6g.large': 0.154,
    'db.m6g.xlarge': 0.308,
  },
  dataTransfer: { internetEgressPerGb: 0.09, egressGbPerCapture: 0.0005 },
  observability: { logsPerGb: 0.5, logMbPerUserMonth: 5, metricsFixedMonthly: 40 },
  compressionRatio: 5,
  fixedMonthly: 110,
};

const DAYS_PER_MONTH = 30;
const HOURS_PER_MONTH = 730;
const BYTES_PER_GB_IN_MB = 1024;
const SNAPSHOT_MONTHS = 24;

export function computeMonthlyCost(
  inputs: CostInputs,
  pricing: Pricing,
  scenario: ScenarioOptions = DEFAULT_SCENARIO,
): CostBreakdown {
  const normalizedInputs = normalizeInputs(inputs);
  const effectiveScenario = {
    tiering: scenario.tiering && normalizedInputs.s3TieringEnabled,
    spot: scenario.spot,
    compression: scenario.compression,
  };
  const categories = computeCategories(normalizedInputs, pricing, effectiveScenario, SNAPSHOT_MONTHS);
  const monthlyCost = sumCategories(categories);
  const ingestGbPerDay = dailyIngestGb(normalizedInputs, pricing, effectiveScenario);
  const projections = Array.from({ length: SNAPSHOT_MONTHS }, (_, index) => {
    const projectionMonth = index + 1;
    const projectionCategories = computeCategories(normalizedInputs, pricing, effectiveScenario, projectionMonth);
    return {
      month: projectionMonth,
      ...projectionCategories,
      total: sumCategories(projectionCategories),
    };
  });

  return {
    inputs: normalizedInputs,
    scenario: effectiveScenario,
    monthlyCost,
    perUserMonthlyCost: monthlyCost / normalizedInputs.dau,
    ingestGbPerDay,
    categories,
    projections,
  };
}

export function computeScenarioDeltas(inputs: CostInputs, pricing: Pricing) {
  const base = computeMonthlyCost(inputs, pricing, {
    tiering: inputs.s3TieringEnabled,
    spot: true,
    compression: true,
  });

  return {
    base,
    withoutTiering: computeMonthlyCost(inputs, pricing, { tiering: false, spot: true, compression: true }),
    withoutSpot: computeMonthlyCost(inputs, pricing, { tiering: inputs.s3TieringEnabled, spot: false, compression: true }),
    withoutCompression: computeMonthlyCost(inputs, pricing, { tiering: inputs.s3TieringEnabled, spot: true, compression: false }),
  };
}

function computeCategories(
  inputs: CostInputs,
  pricing: Pricing,
  scenario: ScenarioOptions,
  month: number,
): CostCategoryBreakdown {
  const capturesPerMonth = monthlyCaptures(inputs);
  const fargateArmMultiplier = 1 - pricing.fargate.armDiscountPercent / 100;
  const vcpuRate = pricing.fargate.vcpuHour * fargateArmMultiplier;
  const gbRate = pricing.fargate.gbHour * fargateArmMultiplier;

  const inferenceHours = (capturesPerMonth * inputs.inferenceDurationSeconds) / 3600;
  const inferenceMemoryGb = inputs.inferenceFargateVcpu * pricing.fargate.inferenceMemoryGbPerVcpu;
  const inference = inferenceHours * (inputs.inferenceFargateVcpu * vcpuRate + inferenceMemoryGb * gbRate);

  const processorHours = (capturesPerMonth * pricing.fargate.processorDurationSeconds) / 3600;
  const processorMemoryGb = inputs.processorFargateVcpu * pricing.fargate.processorMemoryGbPerVcpu;
  const spotMultiplier = scenario.spot ? 1 - inputs.spotDiscountPercent / 100 : 1;
  const processor = processorHours * (inputs.processorFargateVcpu * vcpuRate + processorMemoryGb * gbRate) * spotMultiplier;

  const storage = computeStorage(dailyIngestGb(inputs, pricing, scenario), scenario, month, inputs);
  const dataTransferGb = capturesPerMonth * pricing.dataTransfer.egressGbPerCapture;
  const dataTransfer = dataTransferGb * pricing.dataTransfer.internetEgressPerGb;
  const rds = pricing.rds[inputs.rdsInstanceClass] * HOURS_PER_MONTH * (inputs.readReplica ? 2 : 1);
  const sqs = (capturesPerMonth * 3 * pricing.sqs.requestPerMillion) / 1_000_000;
  const logGb = (inputs.dau * pricing.observability.logMbPerUserMonth) / BYTES_PER_GB_IN_MB;
  const observability = logGb * pricing.observability.logsPerGb + pricing.observability.metricsFixedMonthly;
  const fixed = pricing.fixedMonthly + (capturesPerMonth / 1000) * pricing.s3.putPer1000;

  return {
    inference,
    processor,
    storageStandard: storage.standardGb * pricing.s3.standardGbMonth,
    storageIa: storage.iaGb * pricing.s3.standardIaGbMonth,
    storageGlacier: storage.glacierGb * pricing.s3.glacierIrGbMonth,
    dataTransfer,
    rds,
    sqs,
    observability,
    fixed,
  };
}

function computeStorage(
  dailyGb: number,
  scenario: ScenarioOptions,
  month: number,
  inputs: CostInputs,
): StorageBreakdown {
  const storedDays = Math.max(0, Math.round(month * DAYS_PER_MONTH));
  const tierDays = scenario.tiering
    ? splitTierDays(storedDays, inputs.standardToIaDays, inputs.iaToGlacierDays)
    : { standard: storedDays, ia: 0, glacier: 0 };
  const standardGb = dailyGb * tierDays.standard;
  const iaGb = dailyGb * tierDays.ia;
  const glacierGb = dailyGb * tierDays.glacier;

  return { standardGb, iaGb, glacierGb };
}

function splitTierDays(storedDays: number, standardCutover: number, glacierCutover: number) {
  const standardWindow = clamp(standardCutover, 0, storedDays);
  const glacierStart = Math.max(standardWindow, glacierCutover);
  const iaWindow = clamp(glacierStart - standardWindow, 0, Math.max(0, storedDays - standardWindow));
  const glacierWindow = Math.max(0, storedDays - standardWindow - iaWindow);
  return { standard: standardWindow, ia: iaWindow, glacier: glacierWindow };
}

function dailyIngestGb(inputs: CostInputs, pricing: Pricing, scenario: ScenarioOptions) {
  return (inputs.dau * inputs.capturesPerUserPerDay * effectivePayloadMb(inputs, pricing, scenario)) / BYTES_PER_GB_IN_MB;
}

function effectivePayloadMb(inputs: CostInputs, pricing: Pricing, scenario: ScenarioOptions) {
  return scenario.compression ? inputs.avgPayloadMb / pricing.compressionRatio : inputs.avgPayloadMb;
}

function monthlyCaptures(inputs: CostInputs) {
  return inputs.dau * inputs.capturesPerUserPerDay * DAYS_PER_MONTH;
}

function sumCategories(categories: CostCategoryBreakdown) {
  return Object.values(categories).reduce((sum, value) => sum + value, 0);
}

function normalizeInputs(inputs: CostInputs): CostInputs {
  return {
    ...inputs,
    dau: clamp(inputs.dau, 1, 1_000_000),
    capturesPerUserPerDay: clamp(inputs.capturesPerUserPerDay, 1, 10),
    avgPayloadMb: clamp(inputs.avgPayloadMb, 1, 20),
    inferenceDurationSeconds: clamp(inputs.inferenceDurationSeconds, 1, 60),
    inferenceFargateVcpu: clamp(inputs.inferenceFargateVcpu, 0.25, 16),
    processorFargateVcpu: clamp(inputs.processorFargateVcpu, 0.25, 4),
    spotDiscountPercent: clamp(inputs.spotDiscountPercent, 0, 100),
    standardToIaDays: clamp(inputs.standardToIaDays, 0, 365),
    iaToGlacierDays: clamp(inputs.iaToGlacierDays, 0, 365),
  };
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(Number.isFinite(value) ? value : min, min), max);
}
