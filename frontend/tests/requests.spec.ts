import { test, expect } from "@playwright/test";
test.beforeEach(async ({page}) => { await page.route("https://st.max.ru/js/max-web-app.js", route => route.abort()); });
test("resident creates an issue with a photo and follows status history", async ({
  page,
}) => {
  await page.goto("/");
  await page.screenshot({
    path: "test-results/home-mobile.png",
    fullPage: true,
  });
  await page
    .getByRole("link", { name: "Создать заявку", exact: false })
    .first()
    .click();
  await page
    .getByLabel("Описание")
    .fill("Течёт труба в подъезде — браузерная проверка");
  await page.getByLabel("Фотография").setInputFiles({
    name: "test.png",
    mimeType: "image/png",
    buffer: Buffer.from(
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aB1sAAAAASUVORK5CYII=",
      "base64",
    ),
  });
  await page.getByRole("button", { name: "Продолжить" }).click();
  await expect(
    page.getByRole("heading", { name: "Заявка сохранена" }),
  ).toBeVisible();
  await expect(page.getByText("PIPE_LEAK", { exact: true })).toBeVisible();
  await expect(page.getByRole("img", { name: "Фото к заявке" })).toBeVisible();
  await expect(
    page.getByRole("img", { name: "Фото к заявке" }),
  ).toHaveJSProperty("naturalWidth", 1);
  await page.screenshot({
    path: "test-results/request-mobile.png",
    fullPage: true,
  });
  await page.reload();
  await expect(page.getByRole("img", { name: "Фото к заявке" })).toBeVisible();
  await page.getByRole("button", { name: "Следующий статус" }).click();
  await expect(page.locator(".timeline")).toContainText("Отправлено");
  await page.getByRole("link", { name: "← Мои заявки" }).click();
  await expect(
    page
      .getByRole("heading", {
        name: "Течёт труба в подъезде — браузерная проверка",
      })
      .first(),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});
test("question and application forms are accessible", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Создать заявку" }).click();
  await page.getByLabel("Тип заявки").selectOption("QUESTION");
  await expect(page.getByLabel("Ваш вопрос")).toBeVisible();
  await page.goto("/requests/new?kind=APPLICATION");
  await expect(
    page.getByRole("heading", { name: "Новая заявка" }),
  ).toBeVisible();
});
test("server error is shown without losing input", async ({ page }) => {
  await page.goto("/requests/new");
  await page.getByLabel("Описание").fill("Не работает лифт");
  await page.route("**/api/requests", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: "Сервер временно недоступен" }),
    }),
  );
  await page.getByRole("button", { name: "Продолжить" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "Сервер временно недоступен",
  );
  await expect(page.getByLabel("Описание")).toHaveValue("Не работает лифт");
  await expect(page.getByRole("button", { name: "Продолжить" })).toBeEnabled();
});

test("emergency request is processed by UK and its comments reach the resident", async ({
  page,
}) => {
  const description = `Экстренно течёт труба — ${Date.now()}`;
  await page.goto("/requests/new");
  await page.getByLabel("Тип заявки").selectOption("EMERGENCY");
  await expect(
    page.getByText("В демо она не вызывает аварийную службу.", {
      exact: false,
    }),
  ).toBeVisible();
  await page.getByLabel("Описание").fill(description);
  await page.getByRole("button", { name: "Продолжить" }).click();
  await expect(
    page.getByRole("heading", { name: "Заявка сохранена" }),
  ).toBeVisible();
  const residentURL = page.url();
  await page.getByRole("link", { name: "КАБИНЕТ УК", exact: true }).click();
  await page.getByLabel("Тип заявки").selectOption("EMERGENCY");
  await page.getByRole("combobox", { name: "Организация", exact: true }).selectOption("1");
  await page.getByRole("combobox", { name: "Статус", exact: true }).selectOption("CREATED");
  await expect(page.locator(".request-card").first()).toContainText(
    description,
  );
  await page.screenshot({
    path: "test-results/admin-mobile.png",
    fullPage: true,
  });
  await page.getByRole("heading", { name: description, exact: true }).click();
  await page.getByLabel("Новый статус").selectOption("ACCEPTED");
  await page
    .getByLabel("Комментарий для жителя", { exact: false })
    .fill("Диспетчер принял заявку, мастер назначен.");
  await page.getByRole("button", { name: "Сохранить статус" }).click();
  await expect(page.locator(".timeline")).toContainText(
    "Диспетчер принял заявку, мастер назначен.",
  );
  await page.goto(residentURL);
  await expect(page.locator(".badge")).toHaveText("Принято");
  await expect(page.locator(".timeline")).toContainText(
    "Диспетчер принял заявку, мастер назначен.",
  );
  await page.reload();
  await expect(page.locator(".timeline")).toContainText("УК · демо");
});
test("house profile displays data and works at narrow and desktop widths", async ({
  page,
}) => {
  await page.goto("/house");
  await expect(
    page.getByRole("heading", { name: "Мой дом", exact: true }),
  ).toBeVisible();
  await expect(page.getByText("Общая площадь", { exact: true })).toBeVisible();
  await expect(page.getByText("12 480 м²", { exact: true })).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "УК «Тестовая»" }),
  ).toBeVisible();
  await page.screenshot({
    path: "test-results/house-mobile.png",
    fullPage: true,
  });
  for (const width of [320, 1280]) {
    await page.setViewportSize({ width, height: 900 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  }
  await page.getByRole("link", { name: "Создать заявку" }).click();
  await expect(page.getByLabel("Тип заявки")).toHaveValue("APPLICATION");
});
