export function supplyLabelKey(clientID: string | null | undefined): string {
  if (!clientID) return 'business.supply.unknown';
  return clientID.startsWith('direct:') ? 'business.supply.direct' : 'business.supply.personal';
}

export interface UsagePricing {
  InputTokens: number;
  OutputTokens: number;
  CachedTokens?: number;
  IPPM: number;
  OPPM: number;
  CIPPM?: number;
  Cost?: number;
}

export function usageCostBreakdown(record: UsagePricing) {
  const cached = record.CachedTokens || 0;
  const uncachedInput = ((record.InputTokens - cached) * record.IPPM) / 1_000_000;
  const cachedInput = (cached * (record.CIPPM || 0)) / 1_000_000;
  const output = (record.OutputTokens * record.OPPM) / 1_000_000;
  const total = record.Cost ?? Math.max(0, uncachedInput + cachedInput + output);
  return { uncachedInput, cachedInput, output, total };
}