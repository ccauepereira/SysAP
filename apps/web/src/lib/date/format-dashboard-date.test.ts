import { describe, expect, it } from "vitest";
import {
  formatDashboardDate,
  formatDashboardMachineDate,
} from "./format-dashboard-date";

describe("formatDashboardDate", () => {
  it("formats the presentation date in America/Fortaleza", () => {
    const utcInstant = new Date("2025-06-05T01:30:00.000Z");

    expect(formatDashboardDate(utcInstant)).toBe(
      "Quarta-feira, 04 de junho de 2025",
    );
    expect(formatDashboardMachineDate(utcInstant)).toBe("2025-06-04");
  });
});
