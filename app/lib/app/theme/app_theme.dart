import 'package:flutter/material.dart';

abstract final class AppColors {
  static const canvas = Color(0xFFF4F7F6);
  static const surface = Color(0xFFFCFDFD);
  static const surfaceMuted = Color(0xFFEEF4F2);
  static const ink = Color(0xFF192522);
  static const inkMuted = Color(0xFF66736F);
  static const border = Color(0xFFD9E3E0);
  static const teal = Color(0xFF167C72);
  static const tealSoft = Color(0xFFDDEFEB);
  static const coral = Color(0xFFCA674E);
  static const coralSoft = Color(0xFFF8E6E0);
  static const amber = Color(0xFFB47A24);
  static const amberSoft = Color(0xFFF7EBD6);
  static const navy = Color(0xFF294A5A);
  static const navySoft = Color(0xFFE0ECF1);
}

abstract final class AppTheme {
  static ThemeData light() {
    final scheme =
        ColorScheme.fromSeed(
          seedColor: AppColors.teal,
          brightness: Brightness.light,
          surface: AppColors.surface,
        ).copyWith(
          primary: AppColors.teal,
          secondary: AppColors.coral,
          tertiary: AppColors.amber,
          outline: AppColors.border,
        );
    final base = ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: AppColors.canvas,
      fontFamily: 'Segoe UI',
    );
    return base.copyWith(
      appBarTheme: const AppBarTheme(
        elevation: 0,
        backgroundColor: AppColors.surface,
        foregroundColor: AppColors.ink,
        surfaceTintColor: Colors.transparent,
      ),
      cardTheme: const CardThemeData(
        elevation: 0,
        color: AppColors.surface,
        surfaceTintColor: Colors.transparent,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.all(Radius.circular(8)),
          side: BorderSide(color: AppColors.border),
        ),
      ),
      dividerTheme: const DividerThemeData(color: AppColors.border, space: 1),
      inputDecorationTheme: const InputDecorationTheme(
        filled: true,
        fillColor: AppColors.surface,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.all(Radius.circular(6)),
          borderSide: BorderSide(color: AppColors.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.all(Radius.circular(6)),
          borderSide: BorderSide(color: AppColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.all(Radius.circular(6)),
          borderSide: BorderSide(color: AppColors.teal, width: 1.5),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(6)),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(6)),
          side: const BorderSide(color: AppColors.border),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        ),
      ),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((states) {
          return states.contains(WidgetState.selected) ? Colors.white : null;
        }),
        trackColor: WidgetStateProperty.resolveWith((states) {
          return states.contains(WidgetState.selected)
              ? AppColors.teal
              : AppColors.border;
        }),
      ),
      textTheme: base.textTheme.copyWith(
        headlineMedium: base.textTheme.headlineMedium?.copyWith(
          color: AppColors.ink,
          fontWeight: FontWeight.w700,
        ),
        titleLarge: base.textTheme.titleLarge?.copyWith(
          color: AppColors.ink,
          fontWeight: FontWeight.w700,
        ),
        titleMedium: base.textTheme.titleMedium?.copyWith(
          color: AppColors.ink,
          fontWeight: FontWeight.w700,
        ),
        bodyMedium: base.textTheme.bodyMedium?.copyWith(color: AppColors.ink),
        bodySmall: base.textTheme.bodySmall?.copyWith(
          color: AppColors.inkMuted,
        ),
      ),
    );
  }
}
