import { add, processData, DataProcessor, categorize } from "./utils";

describe("add", () => {
  it("should add two numbers", () => {
    expect(add(1, 2)).toBe(3);
    expect(add(0, 0)).toBe(0);
    expect(add(-1, 1)).toBe(0);
  });

  it("should handle negative numbers", () => {
    expect(add(-5, -3)).toBe(-8);
  });
});

describe("processData", () => {
  test("should process string items", () => {
    const result = processData(["hello", "world"], "sort");
    expect(result).toEqual(["HELLO", "WORLD"]);
  });

  test("should handle empty strings", () => {
    const result = processData(["", "test"], "sort");
    expect(result.length).toBe(2);
  });

  test("should process numeric items", () => {
    const result = processData([42, 200], "sort");
    expect(result).toEqual(["42", "OUT_OF_RANGE"]);
  });
});

describe("DataProcessor", () => {
  it("should process items above threshold", () => {
    const dp = new DataProcessor(3);
    dp.addItem("hi");
    dp.addItem("hello");
    dp.addItem("world");
    const result = dp.process();
    expect(result).toEqual(["hello", "world"]);
  });

  it("should compute stats correctly", () => {
    const dp = new DataProcessor(0);
    dp.addItem("abc");
    dp.addItem("de");
    const stats = dp.getStats();
    expect(stats.count).toBe(2);
    expect(stats.avgLength).toBe(2.5);
  });
});

describe("categorize", () => {
  it("should categorize values correctly", () => {
    expect(categorize(-5)).toBe("negative");
    expect(categorize(0)).toBe("zero");
    expect(categorize(5)).toBe("small");
    expect(categorize(100)).toBe("large");
  });
});
