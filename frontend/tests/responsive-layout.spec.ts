import { expect, test } from "@playwright/test";

const requestFixture = ["1", "2"].map((id) => ({
  id,
  userId: "1",
  houseId: "1",
  description: `Тестовая заявка ${id}`,
  problemType: "OTHER",
  responsibleOrganizationId: "1",
  responsibleOrganization: "УК «Тестовая»",
  status: "CREATED",
  deadline: "2026-09-20T12:00:00Z",
  createdAt: "2026-09-17T12:00:00Z",
  address: "г. Москва, ул. Тестовая, д. 1",
  kind: "APPLICATION",
  text: `Текст тестовой заявки ${id}`,
  hasPhoto: false,
}));

test.beforeEach(async ({ page }) => {
  await page.route("https://st.max.ru/js/max-web-app.js", (route) =>
    route.abort(),
  );
});

test("desktop uses a wide workspace and multi-column content", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto("/max/");

  const main = await page.locator("main").boundingBox();
  expect(main?.width).toBeGreaterThan(1000);

  const steps = await page.locator(".how > div").evaluateAll((items) =>
    items.map((item) => {
      const box = item.getBoundingClientRect();
      return { x: box.x, y: box.y };
    }),
  );
  expect(steps).toHaveLength(3);
  expect(new Set(steps.map(({ y }) => Math.round(y))).size).toBe(1);
  expect(steps[1].x).toBeGreaterThan(steps[0].x);
  expect(steps[2].x).toBeGreaterThan(steps[1].x);

  await page.route("**/max/api/requests", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(requestFixture),
    }),
  );
  await page.goto("/max/requests");
  const cards = await page.locator(".request-card").evaluateAll((items) =>
    items.map((item) => {
      const box = item.getBoundingClientRect();
      return { x: box.x, y: box.y };
    }),
  );
  expect(cards).toHaveLength(2);
  expect(Math.round(cards[0].y)).toBe(Math.round(cards[1].y));
  expect(cards[1].x).toBeGreaterThan(cards[0].x);
});

test("mobile keeps its existing compact layout", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/max/");

  const mobile = await page.evaluate(() => {
    const main = document.querySelector("main")!;
    const heading = document.querySelector(".hero h1")!;
    const nav = document.querySelector(".bottom-nav")!;
    return {
      mainPaddingLeft: getComputedStyle(main).paddingLeft,
      headingSize: getComputedStyle(heading).fontSize,
      navPosition: getComputedStyle(nav).position,
      navBottom: getComputedStyle(nav).bottom,
    };
  });

  expect(mobile).toEqual({
    mainPaddingLeft: "20px",
    headingSize: "36px",
    navPosition: "fixed",
    navBottom: "0px",
  });
  await expect(page.locator(".how > div")).toHaveCount(3);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});
