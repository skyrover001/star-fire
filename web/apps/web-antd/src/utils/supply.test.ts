import { describe, expect, it } from 'vitest';

import { supplyLabelKey, usageCostBreakdown } from './supply';

describe('supply display', () => {
  it('distinguishes direct, personal and unknown sources', () => {
    expect(supplyLabelKey('direct:backend')).toBe('business.supply.direct');
    expect(supplyLabelKey('client')).toBe('business.supply.personal');
    expect(supplyLabelKey('')).toBe('business.supply.unknown');
    expect(supplyLabelKey(undefined)).toBe('business.supply.unknown');
  });

  it('keeps cached pricing separate and uses the recorded charge', () => {
    const record = { InputTokens: 10, CachedTokens: 4, OutputTokens: 2, IPPM: 2, OPPM: 6, CIPPM: 0.5 };
    expect(usageCostBreakdown(record)).toMatchObject({ uncachedInput: 0.000012, cachedInput: 0.000002, output: 0.000012 });
    expect(usageCostBreakdown(record).total).toBeCloseTo(0.000026, 12);
    expect(usageCostBreakdown({ ...record, Cost: 0 }).total).toBe(0);
    expect(usageCostBreakdown({ ...record, Cost: 0.01 }).total).toBe(0.01);
  });
});